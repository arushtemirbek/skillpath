package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/arushtemirbek/skillpath/internal/config"
	"github.com/arushtemirbek/skillpath/internal/logger"
	"github.com/arushtemirbek/skillpath/internal/server"
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
	handler := server.NewRouter(cfg.HTTP, log)
	return server.New(cfg.HTTP, handler, log).Run(ctx)
}
