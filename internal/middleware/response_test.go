package middleware

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseStatus(t *testing.T) {
	for _, implicit := range []bool{false, true} {
		recorder := httptest.NewRecorder()
		writer := &responseWriter{ResponseWriter: recorder, statusCode: http.StatusOK}
		if implicit {
			_, _ = writer.Write([]byte("ok"))
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
		first := recorder.Code
		writer.WriteHeader(http.StatusInternalServerError)
		if writer.statusCode != first || recorder.Code != first {
			t.Fatal("second status replaced first status")
		}
	}
}

func TestResponseControllerFlush(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &responseWriter{ResponseWriter: recorder}
	if err := http.NewResponseController(writer).Flush(); err != nil {
		t.Fatal(err)
	}
	if !recorder.Flushed || !writer.committed || writer.statusCode != http.StatusOK {
		t.Fatal("flush did not commit status")
	}
}

func TestRecoveryBeforeResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	Recoverer(slog.New(slog.NewJSONHandler(io.Discard, nil)))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })).ServeHTTP(recorder, httptest.NewRequest("GET", "/", nil))
	if recorder.Code != 500 || recorder.Body.String() != "{\"error\":{\"code\":\"INTERNAL_ERROR\",\"message\":\"internal server error\"}}\n" {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRecoveryAfterResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	defer func() {
		value := recover()
		err, ok := value.(error)
		if !ok || !errors.Is(err, http.ErrAbortHandler) {
			t.Fatalf("panic = %v", value)
		}
		if recorder.Body.String() != "partial" || recorder.Code != 200 {
			t.Fatal("second response appended")
		}
	}()
	Recoverer(slog.New(slog.NewJSONHandler(io.Discard, nil)))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("partial"))
		panic("boom")
	})).ServeHTTP(recorder, httptest.NewRequest("GET", "/", nil))
	t.Fatal("expected abort")
}
