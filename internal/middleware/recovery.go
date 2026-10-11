package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/arushtemirbek/skillpath/internal/httperror"
)

func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			defer func() {
				if value := recover(); value != nil {
					if value == http.ErrAbortHandler {
						panic(value)
					}
					log.Error("http handler panic", "panic", value, "request_id", GetRequestID(r.Context()), "stack", string(debug.Stack()))
					if writer.committed {
						// Abort the partially written response; a second status is impossible.
						panic(http.ErrAbortHandler)
					}
					httperror.Write(writer, http.StatusInternalServerError, httperror.InternalError, "internal server error")
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
