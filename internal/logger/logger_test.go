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

// TestNewZapLogger тестирует создание zap логгера.
func TestNewZapLogger(t *testing.T) {
	logger, err := NewZapLogger()
	if err != nil {
		t.Fatalf("NewZapLogger вернул ошибку: %v", err)
	}

	if logger == nil {
		t.Fatal("NewZapLogger не должен возвращать nil")
	}

	logger.Info("test message")
}

// TestNewZapLoggerMultipleCalls тестирует множественные вызовы NewZapLogger.
func TestNewZapLoggerMultipleCalls(t *testing.T) {
	logger1, err1 := NewZapLogger()
	if err1 != nil {
		t.Fatalf("Первый вызов NewZapLogger вернул ошибку: %v", err1)
	}

	logger2, err2 := NewZapLogger()
	if err2 != nil {
		t.Fatalf("Второй вызов NewZapLogger вернул ошибку: %v", err2)
	}

	if logger1 == nil || logger2 == nil {
		t.Fatal("Логгеры не должны быть nil")
	}

	// Проверяем, что логгеры работают независимо
	logger1.Info("message from logger1")
	logger2.Info("message from logger2")
}
