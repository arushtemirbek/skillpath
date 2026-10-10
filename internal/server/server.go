package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/arushtemirbek/skillpath/internal/config"
	"github.com/arushtemirbek/skillpath/internal/server/health"
	appmiddleware "github.com/arushtemirbek/skillpath/internal/middleware"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	router *chi.Mux
	addr   string
}

func New(cfg *config.Config) *Server {
	router := chi.NewRouter()

	router.Use(appmiddleware.RequestID)
	router.Use(appmiddleware.Logging)

	healthHandler := health.NewHandler(time.Now())

	router.Get("/health", healthHandler.Health)

	addr := ":" + cfg.HTTP.Port

	return &Server{
		router: router,
		addr:   addr,
	}
}

func (s *Server) Run() error {
	url := fmt.Sprintf("http://localhost%s", s.addr)

	slog.Info(
		"server starting",
		"url", url,
		"addr", s.addr,
		"health", url+"/health",
	)

	return http.ListenAndServe(s.addr, s.router)
}
