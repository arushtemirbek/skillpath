package httperror

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/arushtemirbek/skillpath/internal/apperror"
)

// WriteError maps application errors, including wrapped errors, to HTTP responses.
func WriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperror.ErrGoalRequired):
		Write(w, http.StatusBadRequest, InvalidRequest, apperror.ErrGoalRequired.Error())
	case errors.Is(err, apperror.ErrInvalidRequest):
		Write(w, http.StatusBadRequest, InvalidRequest, apperror.ErrInvalidRequest.Error())
	case errors.Is(err, apperror.ErrUnprocessableEntity):
		Write(w, http.StatusUnprocessableEntity, InvalidRequest, apperror.ErrUnprocessableEntity.Error())
	case errors.Is(err, apperror.ErrLLMRequestFailed):
		slog.Error("LLM request failed", "error", err)
		Write(w, http.StatusBadGateway, LLMRequestFailed, apperror.ErrLLMRequestFailed.Error())
	case errors.Is(err, apperror.ErrLLMInvalidResponse):
		slog.Error("invalid LLM response", "error", err)
		Write(w, http.StatusBadGateway, LLMInvalidResponse, apperror.ErrLLMInvalidResponse.Error())
	default:
		slog.Error("application error", "error", err)
		Write(w, http.StatusInternalServerError, InternalError, "internal server error")
	}
}
