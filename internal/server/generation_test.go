package server_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arushtemirbek/skillpath/internal/config"
	"github.com/arushtemirbek/skillpath/internal/llm"
	"github.com/arushtemirbek/skillpath/internal/roadmap"
	"github.com/arushtemirbek/skillpath/internal/server"
)

func TestGenerationThroughHTTPServiceAndProvider(t *testing.T) {
	const good = `{"title":"Go Backend","description":"Overview","sections":[{"title":"Go","description":"Basics","topics":[{"title":"Interfaces"}]}]}`
	for _, tc := range []struct {
		name, input, content           string
		upstreamStatus, expectedStatus int
		expectedCode                   string
		expectedCalls                  int
	}{
		{"success", `{"goal":"Learn Go","context":"Python developer"}`, good, 200, 200, "", 1},
		{"empty goal", `{"goal":" "}`, good, 200, 400, "INVALID_REQUEST", 0},
		{"invalid JSON", `{`, good, 200, 400, "INVALID_REQUEST", 0},
		{"unavailable", `{"goal":"Go"}`, "", 503, 502, "LLM_REQUEST_FAILED", 1},
		{"invalid structure", `{"goal":"Go"}`, `{}`, 200, 502, "LLM_INVALID_RESPONSE", 1},
		{"empty topics", `{"goal":"Go"}`, strings.Replace(good, `[{"title":"Interfaces"}]`, `[]`, 1), 200, 502, "LLM_INVALID_RESPONSE", 1},
		{"nested topic", `{"goal":"Go"}`, strings.Replace(good, `"title":"Interfaces"`, `"title":"Interfaces","children":[]`, 1), 200, 502, "LLM_INVALID_RESPONSE", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := make(chan struct{}, 2)
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls <- struct{}{}
				if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("incorrect provider request")
				}
				w.WriteHeader(tc.upstreamStatus)
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": tc.content}}}})
			}))
			defer provider.Close()
			log := slog.New(slog.NewJSONHandler(io.Discard, nil))
			client, err := llm.NewOpenAI(config.OpenAIConfig{APIKey: "test-key", Model: "test-model", BaseURL: provider.URL, Timeout: time.Second}, log)
			if err != nil {
				t.Fatal(err)
			}
			handler := roadmap.NewHandler(roadmap.NewService(roadmap.NewGenerator(client)), log)
			router := server.NewRouter(config.HTTPConfig{RequestTimeout: time.Second}, log, handler.Generate, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
			request := httptest.NewRequest("POST", "/roadmaps/generate", strings.NewReader(tc.input)).WithContext(context.Background())
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Request-ID", "integration-request")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, request)
			if w.Code != tc.expectedStatus || len(calls) != tc.expectedCalls {
				t.Fatalf("response=%d %s, calls=%d", w.Code, w.Body.String(), len(calls))
			}
			if w.Header().Get("X-Request-ID") != "integration-request" {
				t.Fatal("request ID missing")
			}
			if tc.expectedStatus == 200 {
				var result roadmap.Roadmap
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if err := result.Validate(); err != nil {
					t.Fatal(err)
				}
			} else {
				var response struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Error.Code != tc.expectedCode {
					t.Fatalf("error code=%s", response.Error.Code)
				}
			}
		})
	}
}
