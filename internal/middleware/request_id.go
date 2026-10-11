package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/arushtemirbek/skillpath/internal/httperror"
)

type requestIDContextKey struct{}

const requestIDHeader = "X-Request-ID"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)

		if !validRequestID(requestID) {
			var err error
			requestID, err = generateRequestID()
			if err != nil {
				slog.Error("failed to generate request ID", "error", err)
				httperror.Write(w, http.StatusInternalServerError, httperror.InternalError, "internal server error")
				return
			}
		}

		ctx := context.WithValue(
			r.Context(),
			requestIDContextKey{},
			requestID,
		)

		w.Header().Set(requestIDHeader, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	if !ok {
		return ""
	}

	return requestID
}

func generateRequestID() (string, error) {
	bytes := make([]byte, 8)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, char := range id {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.') {
			return false
		}
	}
	return true
}
