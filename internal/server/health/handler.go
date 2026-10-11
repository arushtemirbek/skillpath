package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Checker can be implemented by OpenAI, database or cache adapters.
// Implementations must honor context cancellation.
type Checker interface{ Check(context.Context) error }

type Dependency struct {
	Name    string
	Checker Checker
}

type Handler struct {
	log          *slog.Logger
	timeout      time.Duration
	dependencies []Dependency
}

func NewHandler(log *slog.Logger, timeout time.Duration, dependencies ...Dependency) *Handler {
	seen := make(map[string]bool)
	for _, dependency := range dependencies {
		if dependency.Name == "" || dependency.Checker == nil || seen[dependency.Name] {
			panic("health: dependency must have a unique name and a checker")
		}
		seen[dependency.Name] = true
	}
	if timeout <= 0 {
		panic("health: timeout must be positive")
	}
	return &Handler{log: log, timeout: timeout, dependencies: append([]Dependency(nil), dependencies...)}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	type checkResult struct {
		name string
		err  error
	}
	results := make(chan checkResult, len(h.dependencies))
	checks := make(map[string]string, len(h.dependencies))
	for _, dependency := range h.dependencies {
		checks[dependency.Name] = "error"
		go func(d Dependency) {
			// Contain checker panics so they cannot crash the process from this goroutine.
			var err error
			defer func() {
				if recover() != nil {
					results <- checkResult{name: d.Name, err: context.Canceled}
					return
				}
				results <- checkResult{name: d.Name, err: err}
			}()
			err = d.Checker.Check(ctx)
		}(dependency)
	}
	remaining := len(h.dependencies)
	failed := false
	for remaining > 0 {
		select {
		case result := <-results:
			remaining--
			if result.err == nil {
				checks[result.name] = "ok"
			} else {
				failed = true
				// Checker errors can contain credentials: expose only dependency names.
				h.log.Error("dependency health check failed", "dependency", result.name)
			}
		case <-ctx.Done():
			failed = true
			remaining = 0
		}
	}
	status, code := "ok", http.StatusOK
	if failed {
		status, code = "error", http.StatusServiceUnavailable
	}
	response := struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}{Status: status, Checks: checks}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("failed to write health response", "error", err)
	}
}
