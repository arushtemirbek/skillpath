package health

import (
	"log/slog"
	"net/http"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

// Health is a liveness check and does not call external services.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("{\"status\":\"ok\"}\n")); err != nil {
		slog.Error("failed to write health response", "error", err)
	}
}
