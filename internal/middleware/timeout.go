package middleware

import (
	"context"
	"net/http"
	"time"
)

// RequestTimeout sets a deadline for downstream operations. Handlers and
// external clients must honor the request context; it does not kill goroutines.
func RequestTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
