package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func WithLogging(logger *zap.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		respData := &responseData{status: 200}
		lw := &loggingResponseWriter{ResponseWriter: w, responseData: respData}

		h.ServeHTTP(lw, r)

		duration := time.Since(start)
		logger.Info("request completed",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Int("status", respData.status),
			zap.Duration("duration", duration),
			zap.Int("size", respData.size),
		)
	})
}
