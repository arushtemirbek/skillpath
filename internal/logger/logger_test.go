package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewDoesNotChangeDefaultLogger(t *testing.T) {
	defaultLogger := slog.Default()
	var output bytes.Buffer
	log, err := New("info", &output)
	if err != nil {
		t.Fatal(err)
	}
	log.Debug("filtered")
	log.Info("started", "service", "skillpath")
	if slog.Default() != defaultLogger {
		t.Fatal("default logger replaced")
	}
	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["msg"] != "started" || entry["level"] != "INFO" || entry["service"] != "skillpath" {
		t.Fatalf("log entry = %v", entry)
	}
}
