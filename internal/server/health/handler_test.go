package health

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type checkFunc func(context.Context) error

func (f checkFunc) Check(ctx context.Context) error { return f(ctx) }

func TestDependencyHealth(t *testing.T) {
	for _, fail := range []bool{false, true} {
		log := slog.New(slog.NewJSONHandler(io.Discard, nil))
		h := NewHandler(log, time.Second,
			Dependency{Name: "openai", Checker: checkFunc(func(context.Context) error {
				if fail {
					return errors.New("private API key")
				}
				return nil
			})},
			Dependency{Name: "database", Checker: checkFunc(func(context.Context) error { return nil })},
		)
		w := httptest.NewRecorder()
		h.Health(w, httptest.NewRequest("GET", "/health", nil))
		var result struct {
			Status string
			Checks map[string]string
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Checks["database"] != "ok" {
			t.Fatal(result)
		}
		if fail {
			if w.Code != 503 || result.Checks["openai"] != "error" || result.Status != "error" {
				t.Fatal(result)
			}
		} else if w.Code != 200 || result.Checks["openai"] != "ok" || result.Status != "ok" {
			t.Fatal(result)
		}
		if strings.Contains(w.Body.String(), "private") {
			t.Fatal("secret leaked")
		}
	}
}

func TestChecksRunConcurrently(t *testing.T) {
	started := make(chan struct{}, 2)
	check := checkFunc(func(ctx context.Context) error {
		started <- struct{}{}
		for len(started) < 2 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Millisecond):
			}
		}
		return nil
	})
	h := NewHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Second, Dependency{Name: "a", Checker: check}, Dependency{Name: "b", Checker: check})
	w := httptest.NewRecorder()
	h.Health(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code != 200 {
		t.Fatalf("checks ran sequentially: %s", w.Body.String())
	}
}

func TestCheckTimeoutAndPanic(t *testing.T) {
	h := NewHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), 5*time.Millisecond,
		Dependency{Name: "slow", Checker: checkFunc(func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })},
		Dependency{Name: "panic", Checker: checkFunc(func(context.Context) error { panic("boom") })},
	)
	w := httptest.NewRecorder()
	h.Health(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code != 503 {
		t.Fatalf("response=%d %s", w.Code, w.Body.String())
	}
}
