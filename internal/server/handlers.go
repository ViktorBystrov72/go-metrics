package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

// Handlers содержит HTTP обработчики
type Handlers struct {
	storage storage.Storage
}

// NewHandlers создает новые обработчики
func NewHandlers(storage storage.Storage) *Handlers {
	return &Handlers{
		storage: storage,
	}
}

// UpdateHandler обрабатывает POST запросы для обновления метрик
func (h *Handlers) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	if name == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	switch metricType {
	case string(storage.Gauge):
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(name, v)
	case string(storage.Counter):
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(name, v)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, "OK"); err != nil {
		log.Printf("Ошибка при записи ответа в UpdateHandler: %v", err)
	}
}

// ValueHandler обрабатывает GET запросы для получения значений метрик
func (h *Handlers) ValueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	if name == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	switch metricType {
	case string(storage.Gauge):
		value, exists := h.storage.GetGauge(name)
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, value); err != nil {
			log.Printf("Ошибка при записи ответа в ValueHandler (gauge): %v", err)
		}
	case string(storage.Counter):
		value, exists := h.storage.GetCounter(name)
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, value); err != nil {
			log.Printf("Ошибка при записи ответа в ValueHandler (counter): %v", err)
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// IndexHandler обрабатывает GET запросы для отображения HTML-страницы со всеми метриками
func (h *Handlers) IndexHandler(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>Метрики</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric-section { margin-bottom: 30px; }
        .metric-item { margin: 5px 0; padding: 5px; background-color: #f5f5f5; border-radius: 3px; }
        h1 { color: #333; }
        h2 { color: #666; }
    </style>
</head>
<body>
    <h1>Метрики системы</h1>
    
    <div class="metric-section">
        <h2>Gauge метрики</h2>
        {{range $name, $value := .Gauges}}
        <div class="metric-item">
            <strong>{{$name}}:</strong> {{$value}}
        </div>
        {{else}}
        <div class="metric-item">Нет gauge метрик</div>
        {{end}}
    </div>
    
    <div class="metric-section">
        <h2>Counter метрики</h2>
        {{range $name, $value := .Counters}}
        <div class="metric-item">
            <strong>{{$name}}:</strong> {{$value}}
        </div>
        {{else}}
        <div class="metric-item">Нет counter метрик</div>
        {{end}}
    </div>
</body>
</html>`

	tmpl, err := template.New("metrics").Parse(htmlTemplate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   gauges,
		Counters: counters,
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Ошибка при рендеринге шаблона: %v", err)
	}
}

// UpdateJSONHandler обрабатывает POST запросы для обновления метрик в JSON формате
func (h *Handlers) UpdateJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var resp models.Metrics
	resp.ID = m.ID
	resp.MType = m.MType
	switch m.MType {
	case "gauge":
		if m.Value == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(m.ID, *m.Value)
		resp.Value = m.Value
	case "counter":
		if m.Delta == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(m.ID, *m.Delta)
		// Получаем актуальное значение после обновления
		v, ok := h.storage.GetCounter(m.ID)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp.Delta = &v
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Ошибка при кодировании JSON ответа в UpdateJSONHandler: %v", err)
	}
}

// ValueJSONHandler обрабатывает POST запросы для получения значений метрик в JSON формате
func (h *Handlers) ValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var resp models.Metrics
	resp.ID = m.ID
	resp.MType = m.MType
	switch m.MType {
	case "gauge":
		v, ok := h.storage.GetGauge(m.ID)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Value = &v
	case "counter":
		v, ok := h.storage.GetCounter(m.ID)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Delta = &v
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Ошибка при кодировании JSON ответа в ValueJSONHandler: %v", err)
	}
}
