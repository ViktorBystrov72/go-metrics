package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	"github.com/go-chi/chi/v5"
)

// ExampleHandlers_UpdateHandler демонстрирует обновление gauge метрики через URL параметры.
func ExampleHandlers_UpdateHandler() {
	// Создаем хранилище и обработчики без ключа для упрощения
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	// Создаем тестовый запрос для обновления gauge метрики
	req := httptest.NewRequest("POST", "/update/gauge/testMetric/123.45", nil)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"type", "name", "value"},
			Values: []string{"gauge", "testMetric", "123.45"},
		},
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Выполняем запрос
	handlers.UpdateHandler(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Response: %s\n", w.Body.String())

	// Output:
	// Status: 200
	// Response: OK
}

// ExampleHandlers_ValueHandler демонстрирует получение значения метрики.
func ExampleHandlers_ValueHandler() {
	// Создаем хранилище и обработчики
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	// Добавляем тестовую метрику
	storage.UpdateGauge("testMetric", 123.45)

	// Создаем тестовый запрос для получения значения
	req := httptest.NewRequest("GET", "/value/gauge/testMetric", nil)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"type", "name"},
			Values: []string{"gauge", "testMetric"},
		},
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Выполняем запрос
	handlers.ValueHandler(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Value: %s\n", strings.TrimSpace(w.Body.String()))

	// Output:
	// Status: 200
	// Value: 123.45
}

// ExampleHandlers_UpdateJSONHandler демонстрирует обновление метрики в JSON формате.
func ExampleHandlers_UpdateJSONHandler() {
	// Создаем хранилище и обработчики без ключа
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	// Создаем метрику для обновления
	metric := models.Metrics{
		ID:    "testCounter",
		MType: "counter",
		Delta: func() *int64 { v := int64(10); return &v }(),
	}

	// Сериализуем в JSON
	jsonData, _ := json.Marshal(metric)

	// Создаем тестовый запрос
	req := httptest.NewRequest("POST", "/update/", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем запрос
	handlers.UpdateJSONHandler(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	// Проверяем только статус, так как хеш может меняться
	fmt.Printf("Response contains delta: %t\n", strings.Contains(w.Body.String(), "delta"))

	// Output:
	// Status: 200
	// Response contains delta: true
}

// ExampleHandlers_ValueJSONHandler демонстрирует получение значения метрики в JSON формате.
func ExampleHandlers_ValueJSONHandler() {
	// Создаем хранилище и обработчики
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	// Добавляем тестовую метрику
	storage.UpdateGauge("testMetric", 123.45)

	// Создаем запрос для получения значения
	metric := models.Metrics{
		ID:    "testMetric",
		MType: "gauge",
	}

	// Сериализуем в JSON
	jsonData, _ := json.Marshal(metric)

	// Создаем тестовый запрос
	req := httptest.NewRequest("POST", "/value/", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем запрос
	handlers.ValueJSONHandler(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	// Проверяем только статус, так как хеш может меняться
	fmt.Printf("Response contains value: %t\n", strings.Contains(w.Body.String(), "value"))

	// Output:
	// Status: 200
	// Response contains value: true
}

// ExampleHandlers_PingHandler демонстрирует проверку доступности хранилища.
func ExampleHandlers_PingHandler() {
	// Создаем хранилище и обработчики
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	handlers.PingHandler(w, req)

	fmt.Printf("Status: %d\n", w.Code)

	// Output:
	// Status: 200
}

// ExampleHandlers_UpdatesHandler демонстрирует массовое обновление метрик.
func ExampleHandlers_UpdatesHandler() {
	// Создаем хранилище и обработчики без ключа
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	// Создаем массив метрик для обновления
	metrics := []models.Metrics{
		{
			ID:    "gauge1",
			MType: "gauge",
			Value: func() *float64 { v := 123.45; return &v }(),
		},
		{
			ID:    "counter1",
			MType: "counter",
			Delta: func() *int64 { v := int64(10); return &v }(),
		},
	}

	// Сериализуем в JSON
	jsonData, _ := json.Marshal(metrics)

	// Создаем тестовый запрос
	req := httptest.NewRequest("POST", "/updates/", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем запрос
	handlers.UpdatesHandler(w, req)

	fmt.Printf("Status: %d\n", w.Code)

	// Output:
	// Status: 200
}

// ExampleHandlers_IndexHandler демонстрирует отображение HTML-страницы с метриками.
func ExampleHandlers_IndexHandler() {
	// Создаем хранилище и обработчики
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	// Добавляем тестовые метрики
	storage.UpdateGauge("testGauge", 123.45)
	storage.UpdateCounter("testCounter", 10)

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	handlers.IndexHandler(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Content-Type: %s\n", w.Header().Get("Content-Type"))

	// Output:
	// Status: 200
	// Content-Type: text/html
}

// ExampleHandlers_UpdateHandler_withHash демонстрирует обновление метрики с проверкой хеша.
func ExampleHandlers_UpdateHandler_withHash() {
	// Создаем хранилище и обработчики с ключом
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "test-key")

	// Создаем тестовый запрос для обновления counter метрики
	req := httptest.NewRequest("POST", "/update/counter/testCounter/10", bytes.NewReader([]byte("testCounter:counter:10")))

	// Вычисляем хеш для проверки целостности
	data := []byte("testCounter:counter:10")
	hash := utils.CalculateHash(data, "test-key")
	req.Header.Set("HashSHA256", hash)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"type", "name", "value"},
			Values: []string{"counter", "testCounter", "10"},
		},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Выполняем запрос
	handlers.UpdateHandler(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Response: %s\n", w.Body.String())

	// Output:
	// Status: 200
	// Response: OK
}
