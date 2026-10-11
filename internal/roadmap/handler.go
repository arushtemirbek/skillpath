package roadmap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/arushtemirbek/skillpath/internal/httperror"
)

const maxRequestBodyBytes = 1 << 20

type RoadmapService interface {
	Generate(ctx context.Context, goal, userContext string) (*Roadmap, error)
}

type Handler struct {
	service RoadmapService
	log     *slog.Logger
}

func NewHandler(service RoadmapService, log *slog.Logger) *Handler {
	return &Handler{service: service, log: log}
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	if header := r.Header.Get("Content-Type"); header != "" {
		mediaType, _, err := mime.ParseMediaType(header)
		if err != nil || mediaType != "application/json" {
			httperror.Write(w, http.StatusBadRequest, httperror.InvalidRequest, "Content-Type must be application/json")
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request *GenerateRoadmapRequest
	if err := decoder.Decode(&request); err != nil {
		message := "invalid JSON request"
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			message = "request body must not exceed 1 MiB"
		}
		httperror.Write(w, http.StatusBadRequest, httperror.InvalidRequest, message)
		return
	}
	if request == nil {
		httperror.Write(w, http.StatusBadRequest, httperror.InvalidRequest, "request must be a JSON object")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		httperror.Write(w, http.StatusBadRequest, httperror.InvalidRequest, "request must contain a single JSON object")
		return
	}
	result, err := h.service.Generate(r.Context(), request.Goal, request.Context)
	if err != nil {
		httperror.WriteError(w, err)
		return
	}
	data, err := json.Marshal(result)
	if err != nil {
		h.log.Error("failed to encode roadmap", "error", err)
		httperror.Write(w, http.StatusInternalServerError, httperror.InternalError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(append(data, '\n')); err != nil {
		h.log.Error("failed to write roadmap", "error", err)
	}
}
