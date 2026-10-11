package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	committed  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.committed {
		return
	}
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		rw.ResponseWriter.WriteHeader(code)
		return
	}
	rw.committed = true
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.committed {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

func (rw *responseWriter) Unwrap() http.ResponseWriter { return rw.ResponseWriter }

func (rw *responseWriter) FlushError() error {
	if !rw.committed {
		rw.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(rw.ResponseWriter).Flush()
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		writer := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		defer func() {
			duration := time.Since(start)

			slog.Info(
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
