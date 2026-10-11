package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/arushtemirbek/skillpath/internal/config"
	"github.com/arushtemirbek/skillpath/internal/httperror"
	appmiddleware "github.com/arushtemirbek/skillpath/internal/middleware"
	"github.com/arushtemirbek/skillpath/internal/server/health"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	router          *chi.Mux
	addr            string
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

func New(cfg *config.Config) *Server {
	router := chi.NewRouter()

	router.Use(appmiddleware.RequestID)
	router.Use(appmiddleware.Logging)
	router.Use(appmiddleware.Recoverer)
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httperror.Write(w, http.StatusNotFound, httperror.InvalidRequest, "route not found")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httperror.Write(w, http.StatusMethodNotAllowed, httperror.InvalidRequest, "method not allowed")
	})

	healthHandler := health.NewHandler(time.Now())

	router.Get("/health", healthHandler.Health)

	addr := ":" + cfg.HTTP.Port

	return &Server{
		router: router,
		addr:   addr,
		httpServer: &http.Server{
			Addr: addr, Handler: router,
			ReadTimeout:  cfg.HTTP.ReadTimeout,
			WriteTimeout: cfg.HTTP.WriteTimeout,
			IdleTimeout:  cfg.HTTP.IdleTimeout,
		},
		shutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}
}

func (s *Server) Run(ctx context.Context) error {
	url := fmt.Sprintf("http://localhost%s", s.addr)

	slog.Info(
		"server starting",
		"url", url,
		"addr", s.addr,
		"health", url+"/health",
	)

	if ctx.Err() != nil {
		return nil
	}
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- s.httpServer.ListenAndServe() }()
	select {
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			_ = s.httpServer.Close()
			<-serveErrors
			return fmt.Errorf("shutdown server: %w", err)
		}
		err := <-serveErrors
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		slog.Info("server stopped")
		return nil
	}
}
