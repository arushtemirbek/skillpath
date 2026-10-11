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
	cfg, err := config.Load()
	if err != nil {
		slog.Error(
			"failed to load config",
			"error", err,
		)

		os.Exit(1)
	}

	logger.Setup(cfg.Log.Level)
	slog.Info(
		"application starting",
		"log_level", cfg.Log.Level,
	)

	srv := server.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := srv.Run(ctx); err != nil {
		slog.Error(
			"server stopped with error",
			"error", err,
		)

		os.Exit(1)
	}
}
