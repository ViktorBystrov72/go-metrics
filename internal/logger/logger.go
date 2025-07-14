// Package logger предоставляет middleware для логирования HTTP-запросов.
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

// WithLogging создает middleware для логирования HTTP-запросов.
// Логирует URI, метод, статус ответа, время выполнения и размер ответа.
// Использует zap логгер для структурированного логирования.
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

// NewZapLogger создает zap.Logger с оптимизированной конфигурацией для сервиса.
// Можно переиспользовать во всех сервисах проекта.
func NewZapLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Sampling = &zap.SamplingConfig{
		Initial:    100,
		Thereafter: 100,
	}
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	return config.Build()
}
