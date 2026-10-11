package roadmap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/arushtemirbek/skillpath/internal/apperror"
	"github.com/arushtemirbek/skillpath/internal/httperror"
)

func TestHandlerGenerate(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType string
		providerError           error
		status                  int
		code                    string
	}{
		{name: "success", body: `{"goal":"Go","context":"Python developer"}`, status: 200},
		{name: "optional context", body: `{"goal":"Go"}`, contentType: "application/json; charset=utf-8", status: 200},
		{name: "missing goal", body: `{}`, status: 400, code: httperror.InvalidRequest},
		{name: "blank goal", body: `{"goal":" "}`, status: 400, code: httperror.InvalidRequest},
		{name: "invalid JSON", body: `{`, status: 400, code: httperror.InvalidRequest},
		{name: "empty body", body: ``, status: 400, code: httperror.InvalidRequest},
		{name: "null", body: `null`, status: 400, code: httperror.InvalidRequest},
		{name: "array", body: `[]`, status: 400, code: httperror.InvalidRequest},
		{name: "wrong goal type", body: `{"goal":2}`, status: 400, code: httperror.InvalidRequest},
		{name: "wrong context type", body: `{"goal":"Go","context":{}}`, status: 400, code: httperror.InvalidRequest},
		{name: "unknown field", body: `{"goal":"Go","unexpected":true}`, status: 400, code: httperror.InvalidRequest},
		{name: "multiple objects", body: `{"goal":"Go"}{}`, status: 400, code: httperror.InvalidRequest},
		{name: "trailing garbage", body: `{"goal":"Go"} junk`, status: 400, code: httperror.InvalidRequest},
		{name: "oversized body", body: `{"goal":"` + strings.Repeat("a", maxRequestBodyBytes) + `"}`, status: 400, code: httperror.InvalidRequest},
		{name: "wrong content type", body: `{"goal":"Go"}`, contentType: "text/plain", status: 400, code: httperror.InvalidRequest},
		{name: "provider unavailable", body: `{"goal":"Go"}`, providerError: apperror.ErrLLMRequestFailed, status: 502, code: httperror.LLMRequestFailed},
		{name: "invalid LLM result", body: `{"goal":"Go"}`, providerError: apperror.ErrLLMInvalidResponse, status: 502, code: httperror.LLMInvalidResponse},
		{name: "internal failure", body: `{"goal":"Go"}`, providerError: errors.New("private internal details"), status: 500, code: httperror.InternalError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			service := NewService(stubLLM(func(ctx context.Context, goal, userContext string) (*Roadmap, error) {
				calls++
				return validRoadmap(), tc.providerError
			}))
			handler := NewHandler(service, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			request := httptest.NewRequest("POST", "/roadmaps/generate", strings.NewReader(tc.body))
			if tc.contentType != "" {
				request.Header.Set("Content-Type", tc.contentType)
			}
			w := httptest.NewRecorder()
			handler.Generate(w, request)
			if w.Code != tc.status || w.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("response=%d %s", w.Code, w.Body.String())
			}
			if tc.status == 400 && calls != 0 {
				t.Fatal("provider called for invalid input")
			}
			if tc.status == 200 {
				var result Roadmap
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(&result, validRoadmap()) {
					t.Fatalf("result=%+v", result)
				}
			} else {
				var response struct {
					Error httperror.Detail `json:"error"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Error.Code != tc.code || response.Error.Message == "" {
					t.Fatalf("response=%+v", response)
				}
				if strings.Contains(w.Body.String(), "private") {
					t.Fatal("internal details leaked")
				}
				if tc.name == "missing goal" && response.Error.Message != "goal is required" {
					t.Fatal(response.Error.Message)
				}
			}
		})
	}
}
