package logger

import (
	"fmt"
	"io"
	"log/slog"
)

// New constructs a JSON logger without changing the process-global logger.
func New(level string, output io.Writer) (*slog.Logger, error) {
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: logLevel})), nil
}
