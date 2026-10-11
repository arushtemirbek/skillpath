package health

import (
	"encoding/json"
	"github.com/arushtemirbek/skillpath/internal/httperror"
	"log/slog"
	"net/http"
	"time"

	appmiddleware "github.com/arushtemirbek/skillpath/internal/middleware"
)

type Handler struct {
	startedAt time.Time
}

type Response struct {
	Status    string      `json:"status"`
	Service   string      `json:"service"`
	Uptime    string      `json:"uptime"`
	Timestamp time.Time   `json:"timestamp"`
	Request   RequestInfo `json:"request"`
}

type RequestInfo struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

func NewHandler(startedAt time.Time) *Handler {
	return &Handler{
		startedAt: startedAt,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Status:    "ok",
		Service:   "skillpath",
		Uptime:    time.Since(h.startedAt).Round(time.Second).String(),
		Timestamp: time.Now(),
		Request: RequestInfo{
			ID:     appmiddleware.GetRequestID(r.Context()),
			Method: r.Method,
			Path:   r.URL.Path,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		slog.Error("failed to encode health response", "error", err)
		httperror.Write(w, http.StatusInternalServerError, httperror.InternalError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(append(data, '\n')); err != nil {
		slog.Error("failed to write health response", "error", err)
	}
}
