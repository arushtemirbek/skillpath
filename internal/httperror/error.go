package httperror

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

const (
	InvalidRequest     = "INVALID_REQUEST"
	LLMRequestFailed   = "LLM_REQUEST_FAILED"
	LLMInvalidResponse = "LLM_INVALID_RESPONSE"
	InternalError      = "INTERNAL_ERROR"
)

type Detail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Write(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(struct {
		Error Detail `json:"error"`
	}{Error: Detail{Code: code, Message: message}})
	if err != nil {
		slog.Error("failed to write error response", "error", err)
	}
}
