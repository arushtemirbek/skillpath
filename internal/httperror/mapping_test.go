package httperror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arushtemirbek/skillpath/internal/apperror"
)

func TestErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{apperror.ErrGoalRequired, 400, InvalidRequest},
		{apperror.ErrInvalidRequest, 400, InvalidRequest},
		{apperror.ErrUnprocessableEntity, 422, InvalidRequest},
		{apperror.ErrLLMRequestFailed, 502, LLMRequestFailed},
		{apperror.ErrLLMInvalidResponse, 502, LLMInvalidResponse},
		{errors.New("private details"), 500, InternalError},
	} {
		w := httptest.NewRecorder()
		WriteError(w, fmt.Errorf("wrapped: %w", tc.err))
		var response struct {
			Error Detail `json:"error"`
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
