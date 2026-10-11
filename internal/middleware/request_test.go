package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestID(t *testing.T) {
	for _, id := range []string{"", "client-123", "invalid id", strings.Repeat("x", 129)} {
		r := httptest.NewRequest("GET", "/health", nil)
		r.Header.Set(requestIDHeader, id)
		w := httptest.NewRecorder()
		RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actual := GetRequestID(r.Context())
			if actual == "" || actual != w.Header().Get(requestIDHeader) {
				t.Fatal("request ID missing or inconsistent")
			}
			if validRequestID(id) && actual != id {
				t.Fatal("client ID not preserved")
			}
			if !validRequestID(id) && (!validRequestID(actual) || actual == id) {
				t.Fatal("invalid ID not replaced")
			}
		})).ServeHTTP(w, r)
	}
}

func TestRequestTimeout(t *testing.T) {
	var captured context.Context
	RequestTimeout(5*time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Context()
		select {
		case <-r.Context().Done():
			if !errors.Is(r.Context().Err(), context.DeadlineExceeded) {
				t.Fatal(r.Context().Err())
			}
		case <-time.After(time.Second):
			t.Fatal("deadline not enforced")
		}
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if captured.Err() == nil {
		t.Fatal("context still active")
	}
}

func TestRequestTimeoutPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	RequestTimeout(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !errors.Is(r.Context().Err(), context.Canceled) {
			t.Fatal("parent cancellation lost")
		}
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil).WithContext(ctx))
}
