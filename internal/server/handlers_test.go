package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	"github.com/go-chi/chi/v5"
)

// TestNewHandlers тестирует создание обработчиков.
func TestNewHandlers(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	if handlers == nil {
		t.Fatal("NewHandlers не должен возвращать nil")
	}
}

// TestAddHashToResponse тестирует добавление хеша к ответу.
func TestAddHashToResponse(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	w := httptest.NewRecorder()

	// Добавляем хеш к ответу
	handlers.addHashToResponse(w, []byte("test data"))

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestAddHashToMetrics тестирует добавление хеша к метрикам.
func TestAddHashToMetrics(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	metric := models.Metrics{
		ID:    "test",
		MType: "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	// Добавляем хеш
	handlers.addHashToMetrics(&metric)

	if metric.Hash == "" {
		t.Error("Хеш должен быть добавлен к метрике")
	}
}

// TestCheckHash тестирует проверку хеша.
func TestCheckHash(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/test", nil)

	// Проверяем хеш
	valid := handlers.checkHash(req)
	// Ожидаем true, так как без хеша проверка пропускается
	if !valid {
		t.Error("Хеш должен быть валидным без заголовка (проверка пропускается)")
	}
}

// TestCheckJSONHash тестирует проверку хеша JSON.
func TestCheckJSONHash(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Создаем тестовый запрос с пустым телом
	req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("{}")))

	// Проверяем хеш JSON
	valid := handlers.checkJSONHash(req)
	// Ожидаем true, так как без хеша проверка пропускается
	if !valid {
		t.Error("JSON хеш должен быть валидным без заголовка (проверка пропускается)")
	}
}

// TestVerifyMetricHash тестирует верификацию хеша метрики.
func TestVerifyMetricHash(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	metric := models.Metrics{
		ID:    "test",
		MType: "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	// Вычисляем правильный хеш
	expectedHash := utils.CalculateHash([]byte("test:gauge:123.450000"), "test-key")
	metric.Hash = expectedHash

	// Верифицируем хеш
	valid := handlers.verifyMetricHash(metric)
	if !valid {
		t.Error("Хеш метрики должен быть валидным")
	}
}

// TestUpdateHandler тестирует обработчик обновления метрик.
func TestUpdateHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Тестируем обновление gauge
	req := httptest.NewRequest("POST", "/update/gauge/test/123.45", nil)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"type", "name", "value"},
			Values: []string{"gauge", "test", "123.45"},
		},
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handlers.UpdateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}

	// Тестируем обновление counter
	req = httptest.NewRequest("POST", "/update/counter/test/100", nil)
	ctx = context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"type", "name", "value"},
			Values: []string{"counter", "test", "100"},
		},
	})
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	handlers.UpdateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestValueHandler тестирует обработчик получения значений метрик.
func TestValueHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Добавляем тестовую метрику
	storage.UpdateGauge("test", 123.45)

	// Получаем значение
	req := httptest.NewRequest("GET", "/value/gauge/test", nil)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"type", "name"},
			Values: []string{"gauge", "test"},
		},
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handlers.ValueHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestIndexHandler тестирует обработчик главной страницы.
func TestIndexHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Добавляем тестовые метрики
	storage.UpdateGauge("test1", 123.45)
	storage.UpdateCounter("test2", 100)

	// Получаем главную страницу
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handlers.IndexHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestUpdateJSONHandler тестирует JSON обработчик обновления.
func TestUpdateJSONHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	metric := models.Metrics{
		ID:    "test",
		MType: "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	// Сериализуем в JSON
	data, _ := json.Marshal(metric)

	req := httptest.NewRequest("POST", "/update/", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.UpdateJSONHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestValueJSONHandler тестирует JSON обработчик получения значений.
func TestValueJSONHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Добавляем тестовую метрику
	storage.UpdateGauge("test", 123.45)

	metric := models.Metrics{
		ID:    "test",
		MType: "gauge",
	}
	data, _ := json.Marshal(metric)

	req := httptest.NewRequest("POST", "/value/", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.ValueJSONHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestPingHandler тестирует обработчик ping.
func TestPingHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	handlers.PingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestUpdatesHandler тестирует обработчик пакетного обновления.
func TestUpdatesHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Создаем тестовые метрики с хешами
	metrics := []models.Metrics{
		{
			ID:    "test1",
			MType: "gauge",
			Value: func() *float64 { v := 123.45; return &v }(),
			Hash:  utils.CalculateHash([]byte("test1:gauge:123.450000"), "test-key"),
		},
		{
			ID:    "test2",
			MType: "counter",
			Delta: func() *int64 { v := int64(100); return &v }(),
			Hash:  utils.CalculateHash([]byte("test2:counter:100"), "test-key"),
		},
	}

	// Сериализуем в JSON
	data, _ := json.Marshal(metrics)

	req := httptest.NewRequest("POST", "/updates/", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.UpdatesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestUpdateHandlerWithInvalidType тестирует обработчик с невалидным типом метрики.
func TestUpdateHandlerWithInvalidType(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Тестируем с невалидным типом
	req := httptest.NewRequest("POST", "/update/invalid/test/123.45", nil)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"type", "name", "value"},
			Values: []string{"invalid", "test", "123.45"},
		},
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handlers.UpdateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

// TestValueHandlerWithInvalidType тестирует обработчик с невалидным типом.
func TestValueHandlerWithInvalidType(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Тестируем с невалидным типом
	req := httptest.NewRequest("GET", "/value/invalid/test", nil)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"type", "name"},
			Values: []string{"invalid", "test"},
		},
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handlers.ValueHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

// TestUpdateJSONHandlerWithInvalidJSON тестирует обработчик с невалидным JSON.
func TestUpdateJSONHandlerWithInvalidJSON(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Отправляем невалидный JSON
	req := httptest.NewRequest("POST", "/update/", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.UpdateJSONHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

// TestValueJSONHandlerWithInvalidJSON тестирует обработчик с невалидным JSON.
func TestValueJSONHandlerWithInvalidJSON(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Отправляем невалидный JSON
	req := httptest.NewRequest("POST", "/value/", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.ValueJSONHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

// TestUpdatesHandlerWithInvalidJSON тестирует обработчик с невалидным JSON.
func TestUpdatesHandlerWithInvalidJSON(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Отправляем невалидный JSON
	req := httptest.NewRequest("POST", "/updates/", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.UpdatesHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

// TestAddHashToMetricsWithNilValue тестирует добавление хеша с nil значением.
func TestAddHashToMetricsWithNilValue(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	metric := models.Metrics{
		ID:    "test",
		MType: "gauge",
		Value: nil,
	}

	// Добавляем хеш
	handlers.addHashToMetrics(&metric)

	// Хеш не должен быть добавлен для nil значения
	if metric.Hash != "" {
		t.Error("Хеш не должен быть добавлен для nil значения")
	}
}

// TestAddHashToMetricsWithCounter тестирует добавление хеша для counter метрики.
func TestAddHashToMetricsWithCounter(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	metric := models.Metrics{
		ID:    "test",
		MType: "counter",
		Delta: func() *int64 { v := int64(100); return &v }(),
	}

	handlers.addHashToMetrics(&metric)

	if metric.Hash == "" {
		t.Error("Хеш должен быть добавлен к counter метрике")
	}
}

// TestAddHashToMetricsWithoutKey тестирует добавление хеша без ключа.
func TestAddHashToMetricsWithoutKey(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "") // Без ключа

	metric := models.Metrics{
		ID:    "test",
		MType: "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	handlers.addHashToMetrics(&metric)

	if metric.Hash != "" {
		t.Error("Хеш не должен быть добавлен без ключа")
	}
}
