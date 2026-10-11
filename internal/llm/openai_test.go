package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arushtemirbek/skillpath/internal/apperror"
	"github.com/arushtemirbek/skillpath/internal/config"
)

const validContent = `{"title":"Go Backend","description":"Overview","sections":[{"title":"Go","description":"Fundamentals","topics":[{"title":"Interfaces"}]}]}`

func completion(content, reason string) string {
	data, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"finish_reason": reason, "message": map[string]any{"content": content}}}})
	return string(data)
}

func newTestClient(t *testing.T, baseURL string, output io.Writer) *OpenAI {
	t.Helper()
	client, err := NewOpenAI(config.OpenAIConfig{APIKey: "test-secret-key", Model: "test-model", BaseURL: baseURL, Timeout: time.Second}, slog.New(slog.NewJSONHandler(output, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestOpenAIRequestAndSchema(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-secret-key" {
			t.Error("incorrect OpenAI request")
		}
		var request chatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Model != "test-model" || request.ResponseFormat.Type != "json_schema" || !request.ResponseFormat.JSONSchema.Strict {
			t.Error("structured output missing")
		}
		if len(request.Messages) != 2 || request.Messages[0].Role != "system" || request.Messages[1].Role != "user" {
			t.Error("incorrect messages")
			return
		}
		var input struct{ Goal, Context string }
		if err := json.Unmarshal([]byte(request.Messages[1].Content), &input); err != nil {
			t.Error(err)
		}
		if input.Goal != "Learn Go" || input.Context != "Python developer" {
			t.Error("user input not passed")
		}
		var schema map[string]any
		if err := json.Unmarshal(request.ResponseFormat.JSONSchema.Schema, &schema); err != nil {
			t.Error(err)
		}
		if schema["type"] != "object" {
			t.Error("schema not forwarded")
		}

		_, _ = io.WriteString(w, completion(validContent, "stop"))
	}))
	defer upstream.Close()
	for _, baseURL := range []string{upstream.URL, upstream.URL + "/v1/"} {
		var logs bytes.Buffer
		result, err := newTestClient(t, baseURL, &logs).Complete(context.Background(), testRequest("Learn Go", "Python developer"))
		if err != nil || result != validContent {
			t.Fatalf("result=%v error=%v", result, err)
		}
		for _, secret := range []string{"test-secret-key", "Python developer", "Authorization"} {
			if strings.Contains(logs.String(), secret) {
				t.Fatalf("secret logged: %s", secret)
			}
		}
		var entry map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"model", "duration", "status", "error"} {
			if _, exists := entry[key]; !exists {
				t.Errorf("log missing %s", key)
			}
		}
	}
}

func TestOpenAIInvalidResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"unavailable", 503, "private provider details", apperror.ErrLLMRequestFailed},
		{"rate limit", 429, "", apperror.ErrLLMRequestFailed},
		{"unauthorized", 401, "", apperror.ErrLLMRequestFailed},
		{"malformed envelope", 200, "{", apperror.ErrLLMInvalidResponse},
		{"no choices", 200, `{"choices":[]}`, apperror.ErrLLMInvalidResponse},
		{"truncated", 200, completion(validContent, "length"), apperror.ErrLLMInvalidResponse},
		{"filtered", 200, completion(validContent, "content_filter"), apperror.ErrLLMInvalidResponse},
		{"refusal", 200, `{"choices":[{"finish_reason":"stop","message":{"refusal":"private refusal"}}]}`, apperror.ErrLLMInvalidResponse},
		{"response too large", 200, strings.Repeat("x", maxResponseBodyBytes+1), apperror.ErrLLMInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer upstream.Close()
			var logs bytes.Buffer
			_, err := newTestClient(t, upstream.URL, &logs).Complete(context.Background(), testRequest("Go", ""))
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want=%v", err, tc.want)
			}
			if strings.Contains(logs.String(), "private") {
				t.Fatal("provider details logged")
			}
		})
	}
}

type doFunc func(*http.Request) (*http.Response, error)

func (f doFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestOpenAITransportErrorIsSanitized(t *testing.T) {
	var logs bytes.Buffer
	client := newTestClient(t, "https://example.com", &logs)
	client.client = doFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("secret transport info test-secret-key")
	})
	_, err := client.Complete(context.Background(), testRequest("Go", ""))
	if !errors.Is(err, apperror.ErrLLMRequestFailed) || strings.Contains(logs.String(), "test-secret-key") {
		t.Fatalf("error=%v logs=%s", err, logs.String())
	}
}

func TestOpenAICancellation(t *testing.T) {
	entered, canceled := make(chan struct{}), make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(entered)
		select {
		case <-r.Context().Done():
			close(canceled)
		case <-time.After(3 * time.Second):
		}
	}))
	defer upstream.Close()
	client := newTestClient(t, upstream.URL, io.Discard)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := client.Complete(ctx, testRequest("Go", "")); finished <- err }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("upstream request not started")
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, apperror.ErrLLMRequestFailed) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("request not canceled")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("upstream did not observe cancellation")
	}
}

func TestOpenAIRedirectIsNotFollowed(t *testing.T) {
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("redirect was followed") }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	_, err := newTestClient(t, source.URL, io.Discard).Complete(context.Background(), testRequest("Go", ""))
	if !errors.Is(err, apperror.ErrLLMRequestFailed) {
		t.Fatal(err)
	}
}

func testRequest(goal, userContext string) CompletionRequest {
	input, _ := json.Marshal(map[string]string{"goal": goal, "context": userContext})
	return CompletionRequest{
		Messages: []Message{{Role: "system", Content: "Return JSON"}, {Role: "user", Content: string(input)}},
		Schema:   &JSONSchema{Name: "test_schema", Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["answer"],"properties":{"answer":{"type":"string"}}}`)},
	}
}

func TestOpenAICheck(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input chatRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input.ResponseFormat != nil || input.MaxCompletionTokens != 64 || len(input.Messages) != 1 {
			t.Error("health must send a short ordinary completion")
		}
		_, _ = io.WriteString(w, completion("OK", "stop"))
	}))
	defer upstream.Close()
	if err := newTestClient(t, upstream.URL, io.Discard).Check(context.Background()); err != nil {
		t.Fatal(err)
	}
}
