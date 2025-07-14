package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddleware(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})
	req := httptest.NewRequest("GET", "/", strings.NewReader(""))
	req.Header.Set("Accept-Encoding", "gzip")
	rw := httptest.NewRecorder()
	GzipMiddleware(h).ServeHTTP(rw, req)
	if rw.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Ожидался gzip Content-Encoding")
	}
}
