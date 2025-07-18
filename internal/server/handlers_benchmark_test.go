package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

// setupTestRouter создает тестовый роутер с хранилищем в памяти
func setupTestRouter() *Router {
	storage := storage.NewMemStorage()
	return NewRouter(storage, "", "")
}

// BenchmarkRouter_UpdateGauge тестирует производительность обновления gauge метрики
func BenchmarkRouter_UpdateGauge(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/update/gauge/test_metric/123.45", nil)
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}
}

// BenchmarkRouter_UpdateCounter тестирует производительность обновления counter метрики
func BenchmarkRouter_UpdateCounter(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/update/counter/test_counter/10", nil)
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}
}

// BenchmarkRouter_GetValue тестирует производительность получения значения метрики
func BenchmarkRouter_GetValue(b *testing.B) {
	router := setupTestRouter()

	// Предварительно добавляем данные
	req := httptest.NewRequest("POST", "/update/gauge/test_value/123.45", nil)
	w := httptest.NewRecorder()
	router.router.ServeHTTP(w, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"test_value","type":"gauge"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}
}

// BenchmarkRouter_UpdateBatch тестирует производительность batch обновлений
func BenchmarkRouter_UpdateBatch(b *testing.B) {
	router := setupTestRouter()

	// Подготавливаем batch данные
	metrics := []models.Metrics{
		{ID: "metric1", MType: "gauge", Value: func() *float64 { v := 123.45; return &v }()},
		{ID: "metric2", MType: "counter", Delta: func() *int64 { v := int64(10); return &v }()},
		{ID: "metric3", MType: "gauge", Value: func() *float64 { v := 67.89; return &v }()},
	}

	data, _ := json.Marshal(metrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/updates/", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}
}

// BenchmarkRouter_GetAllMetrics тестирует производительность получения всех метрик
func BenchmarkRouter_GetAllMetrics(b *testing.B) {
	router := setupTestRouter()

	// Предварительно добавляем данные
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("POST", "/update/gauge/metric_all/123.45", nil)
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}
}

// BenchmarkRouter_UpdateGaugeWithHash тестирует производительность обновления с хешированием
func BenchmarkRouter_UpdateGaugeWithHash(b *testing.B) {
	router := NewRouter(storage.NewMemStorage(), "test-key", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/update/gauge/test_hash/123.45", nil)
		req.Header.Set("HashSHA256", "test-hash")
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}
}

// BenchmarkRouter_UpdateCounterWithHash тестирует производительность обновления counter с хешированием
func BenchmarkRouter_UpdateCounterWithHash(b *testing.B) {
	router := NewRouter(storage.NewMemStorage(), "test-key", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/update/counter/test_hash_counter/10", nil)
		req.Header.Set("HashSHA256", "test-hash")
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}
}

// BenchmarkRouter_UpdateBatchWithHash тестирует производительность batch обновлений с хешированием
func BenchmarkRouter_UpdateBatchWithHash(b *testing.B) {
	router := NewRouter(storage.NewMemStorage(), "test-key", "")

	metrics := []models.Metrics{
		{ID: "metric1", MType: "gauge", Value: func() *float64 { v := 123.45; return &v }()},
		{ID: "metric2", MType: "counter", Delta: func() *int64 { v := int64(10); return &v }()},
	}

	data, _ := json.Marshal(metrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/updates/", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HashSHA256", "test-hash")
		w := httptest.NewRecorder()
		router.router.ServeHTTP(w, req)
	}
}
