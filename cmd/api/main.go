package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/arushtemirbek/skillpath/internal/config"
	"github.com/arushtemirbek/skillpath/internal/llm"
	"github.com/arushtemirbek/skillpath/internal/logger"
	"github.com/arushtemirbek/skillpath/internal/roadmap"
	"github.com/arushtemirbek/skillpath/internal/server"
	"github.com/arushtemirbek/skillpath/internal/server/health"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log, err := logger.New(cfg.Log.Level, os.Stdout)
	if err != nil {
		return err
	}
	// Keep the default logger for helpers that report response write failures.
	slog.SetDefault(log)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Info("application starting", "log_level", cfg.Log.Level)
	client, err := llm.NewOpenAI(cfg.OpenAI, log)
	if err != nil {
		return err
	}
	service := roadmap.NewService(roadmap.NewGenerator(client))
	roadmapHandler := roadmap.NewHandler(service, log)
	healthHandler := health.NewHandler(log, cfg.HTTP.RequestTimeout,
		health.Dependency{Name: "openai", Checker: client},
	)
	handler := server.NewRouter(cfg.HTTP, log, roadmapHandler.Generate, healthHandler.Health)
	return server.New(cfg.HTTP, handler, log).Run(ctx)
}
