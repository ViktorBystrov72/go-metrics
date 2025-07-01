package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Декомпрессия запроса
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body = struct {
				io.Reader
				io.Closer
			}{gz, r.Body}
			defer gz.Close()
		}

		// Компрессия ответа
		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		shouldCompress := false
		ct := w.Header().Get("Content-Type")
		if ct == "" {
			ct = r.Header.Get("Content-Type")
		}
		if acceptsGzip && (strings.Contains(ct, "application/json") || strings.Contains(ct, "text/html")) {
			shouldCompress = true
		}

		if shouldCompress {
			gz := gzip.NewWriter(w)
			defer gz.Close()
			w.Header().Set("Content-Encoding", "gzip")
			grw := &gzipResponseWriter{ResponseWriter: w, Writer: gz}
			next.ServeHTTP(grw, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
