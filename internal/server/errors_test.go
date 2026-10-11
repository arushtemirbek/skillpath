package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arushtemirbek/skillpath/internal/apperror"
	"github.com/arushtemirbek/skillpath/internal/httperror"
)

func TestErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{apperror.ErrGoalRequired, 400, httperror.InvalidRequest},
		{apperror.ErrInvalidRequest, 400, httperror.InvalidRequest},
		{apperror.ErrUnprocessableEntity, 422, httperror.InvalidRequest},
		{apperror.ErrLLMRequestFailed, 502, httperror.LLMRequestFailed},
		{apperror.ErrLLMInvalidResponse, 502, httperror.LLMInvalidResponse},
		{errors.New("private details"), 500, httperror.InternalError},
	} {
		w := httptest.NewRecorder()
		WriteError(w, fmt.Errorf("wrapped: %w", tc.err))
		var response struct {
			Error httperror.Detail `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != tc.status || response.Error.Code != tc.code || response.Error.Message == "" {
			t.Fatalf("response = %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "private details") {
			t.Fatal("internal details leaked")
		}
	}
}
