package server

import (
	"log/slog"
	"net/http"

	"github.com/arushtemirbek/skillpath/internal/config"
	"github.com/arushtemirbek/skillpath/internal/httperror"
	"github.com/arushtemirbek/skillpath/internal/middleware"
	"github.com/arushtemirbek/skillpath/internal/server/health"
	"github.com/go-chi/chi/v5"
)

func NewRouter(cfg config.HTTPConfig, log *slog.Logger) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logging(log))
	router.Use(middleware.Recoverer(log))
	router.Use(middleware.RequestTimeout(cfg.RequestTimeout))
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httperror.Write(w, http.StatusNotFound, httperror.InvalidRequest, "route not found")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httperror.Write(w, http.StatusMethodNotAllowed, httperror.InvalidRequest, "method not allowed")
	})
	router.Get("/health", health.NewHandler().Health)
	return router
}
