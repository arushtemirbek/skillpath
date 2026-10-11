package middleware

import "net/http"

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
