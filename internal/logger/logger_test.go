package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestWithLogging(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte("ok"))
	})
	req := httptest.NewRequest("GET", "/", nil)
	rw := httptest.NewRecorder()
	WithLogging(logger, h).ServeHTTP(rw, req)
	if rw.Code != 201 {
		t.Errorf("Ожидался статус 201, получен %d", rw.Code)
	}
}
