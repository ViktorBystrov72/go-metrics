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
	"github.com/go-chi/chi/v5"
)

func TestNewHandlers(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")
	if handlers == nil {
		t.Error("NewHandlers() вернул nil")
	}
}

func TestUpdateHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

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
}

func TestValueHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	storage.UpdateGauge("test", 123.45)
	handlers := NewHandlers(storage, "")

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

func TestUpdateJSONHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	metric := models.Metrics{
		ID:    "test",
		MType: "gauge",
		Value: floatPtr(123.45),
	}

	body, _ := json.Marshal(metric)
	req := httptest.NewRequest("POST", "/update/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.UpdateJSONHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

func TestValueJSONHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	storage.UpdateGauge("test", 123.45)
	handlers := NewHandlers(storage, "")

	metric := models.Metrics{
		ID:    "test",
		MType: "gauge",
	}

	body, _ := json.Marshal(metric)
	req := httptest.NewRequest("POST", "/value/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.ValueJSONHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

func TestPingHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	handlers.PingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

func TestUpdatesHandler(t *testing.T) {
	storage := storage.NewMemStorage()
	handlers := NewHandlers(storage, "")

	metrics := []models.Metrics{
		{ID: "test1", MType: "gauge", Value: floatPtr(123.45)},
		{ID: "test2", MType: "counter", Delta: intPtr(10)},
	}

	body, _ := json.Marshal(metrics)
	req := httptest.NewRequest("POST", "/updates/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.UpdatesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

func floatPtr(f float64) *float64 { return &f }
func intPtr(i int64) *int64       { return &i }
