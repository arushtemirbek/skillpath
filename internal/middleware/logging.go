package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			writer := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			defer func() {
				duration := time.Since(start)

				log.Info(
					"http request",
					"request_id", GetRequestID(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"status", writer.statusCode,
					"duration", duration,
				)
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
