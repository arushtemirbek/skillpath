package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/arushtemirbek/skillpath/internal/apperror"
	"github.com/arushtemirbek/skillpath/internal/httperror"
)

// WriteError maps application errors, including wrapped errors, to HTTP responses.
func WriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperror.ErrGoalRequired):
		httperror.Write(w, http.StatusBadRequest, httperror.InvalidRequest, apperror.ErrGoalRequired.Error())
	case errors.Is(err, apperror.ErrInvalidRequest):
		httperror.Write(w, http.StatusBadRequest, httperror.InvalidRequest, apperror.ErrInvalidRequest.Error())
	case errors.Is(err, apperror.ErrUnprocessableEntity):
		httperror.Write(w, http.StatusUnprocessableEntity, httperror.InvalidRequest, apperror.ErrUnprocessableEntity.Error())
	case errors.Is(err, apperror.ErrLLMRequestFailed):
		slog.Error("LLM request failed", "error", err)
		httperror.Write(w, http.StatusBadGateway, httperror.LLMRequestFailed, apperror.ErrLLMRequestFailed.Error())
	case errors.Is(err, apperror.ErrLLMInvalidResponse):
		slog.Error("invalid LLM response", "error", err)
		httperror.Write(w, http.StatusBadGateway, httperror.LLMInvalidResponse, apperror.ErrLLMInvalidResponse.Error())
	default:
		slog.Error("application error", "error", err)
		httperror.Write(w, http.StatusInternalServerError, httperror.InternalError, "internal server error")
	}
}
