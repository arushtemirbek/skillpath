package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arushtemirbek/skillpath/internal/config"
)

func testConfig() config.HTTPConfig {
	return config.HTTPConfig{Port: "8080", ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: time.Second, ShutdownTimeout: time.Second, RequestTimeout: time.Second}
}

func newTestServer() *Server {
	cfg := testConfig()
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return New(cfg, NewRouter(cfg, log, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}), log)
}

func TestHTTPRoutes(t *testing.T) {
	s := newTestServer()
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/health", 200}, {"POST", "/health", 405}, {"GET", "/missing", 404}, {"POST", "/roadmaps/generate", 200},
	} {
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status || w.Header().Get("Content-Type") != "application/json" || w.Header().Get("X-Request-ID") == "" {
			t.Fatalf("response = %d %s", w.Code, w.Body.String())
		}
		if tc.path == "/health" && tc.status == 200 && strings.TrimSpace(w.Body.String()) != `{"status":"ok"}` {
			t.Fatal("health does not match specification")
		}
	}
	if s.httpServer.ReadHeaderTimeout != time.Second || s.httpServer.WriteTimeout != 2*time.Second {
		t.Fatal("HTTP timeouts not applied")
	}
}

func TestGracefulShutdown(t *testing.T) {
	s := newTestServer()
	s.httpServer.Addr = "127.0.0.1:0"
	address := make(chan string, 1)
	s.httpServer.BaseContext = func(l net.Listener) context.Context { address <- l.Addr().String(); return context.Background() }
	entered, release := make(chan struct{}), make(chan struct{})
	s.httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write([]byte("done"))
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- s.Run(ctx) }()
	var addr string
	select {
	case addr = <-address:
	case err := <-finished:
		t.Fatalf("startup failed: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("startup timeout")
	}
	requestDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://" + addr + "/slow")
		if err == nil {
			var body []byte
			body, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil && string(body) != "done" {
				err = errors.New("incomplete response")
			}
		}
		requestDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-finished:
		close(release)
		t.Fatalf("shutdown interrupted active request: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-requestDone; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timeout")
	}
}

func TestListenFailure(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	s := newTestServer()
	s.httpServer.Addr = l.Addr().String()
	if err := s.Run(context.Background()); err == nil {
		t.Fatal("listen failure swallowed")
	}
}
