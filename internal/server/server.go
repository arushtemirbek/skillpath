package server

import (
	"log/slog"
	"net/http"

	"github.com/arushtemirbek/skillpath/internal/health"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	router *chi.Mux
	addr   string
}

func New() *Server {
	router := chi.NewRouter()

	router.Get("/health", health.Handler)

	return &Server{
		router: router,
		addr:   ":8080",
	}
}

func (s *Server) Run() error {
	slog.Info(
		"server starting",
		"url", "http://localhost:8080",
		"addr", s.addr,
		"health", "http://localhost:8080/health",
	)

	return http.ListenAndServe(s.addr, s.router)
}
