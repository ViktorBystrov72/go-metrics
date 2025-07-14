// Package middleware предоставляет HTTP middleware для сжатия и обработки запросов.
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

// GzipMiddleware создает middleware для сжатия HTTP-ответов и распаковки запросов.
// Автоматически сжимает ответы, если клиент поддерживает gzip.
// Распаковывает входящие запросы, если они сжаты gzip.
// Устанавливает заголовок Content-Encoding для сжатых ответов.
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

		// Компрессия ответа (всегда, если клиент поддерживает gzip)
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
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
