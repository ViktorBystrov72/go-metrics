package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	"github.com/go-chi/chi/v5"
)

// Handlers содержит HTTP обработчики
type Handlers struct {
	storage storage.Storage
	key     string
}

// NewHandlers создает новые обработчики
func NewHandlers(storage storage.Storage, key string) *Handlers {
	return &Handlers{
		storage: storage,
		key:     key,
	}
}

// addHashToResponse добавляет хеш в заголовки ответа
func (h *Handlers) addHashToResponse(w http.ResponseWriter, data []byte) {
	if h.key != "" {
		hash := utils.CalculateHash(data, h.key)
		w.Header().Set("HashSHA256", hash)
	}
}

// checkHash проверяет хеш запроса
func (h *Handlers) checkHash(r *http.Request) bool {
	if h.key == "" {
		return true // если ключ не задан, проверка не требуется
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}

	// Восстанавливаем тело запроса для дальнейшего использования
	r.Body = io.NopCloser(bytes.NewReader(body))

	receivedHash := r.Header.Get("HashSHA256")
	if receivedHash == "" {
		return false // если ключ задан, но хеш не передан - ошибка
	}

	return utils.VerifyHash(body, h.key, receivedHash)
}

// UpdateHandler обрабатывает POST запросы для обновления метрик
func (h *Handlers) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !h.checkHash(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
		value, err := h.storage.GetGauge(name)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, value); err != nil {
			log.Printf("Ошибка при записи ответа в ValueHandler (gauge): %v", err)
		}
	case string(storage.Counter):
		value, err := h.storage.GetCounter(name)
		if err != nil {
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
	w.Header().Set("Content-Type", "application/json")

	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !h.checkHash(r) {
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
		v, err := h.storage.GetCounter(m.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp.Delta = &v
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

	responseData, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Ошибка при кодировании JSON ответа в UpdateJSONHandler: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.addHashToResponse(w, responseData)

	if _, err := w.Write(responseData); err != nil {
		log.Printf("Ошибка при записи ответа в UpdateJSONHandler: %v", err)
	}
}

// ValueJSONHandler обрабатывает POST запросы для получения значений метрик в JSON формате
func (h *Handlers) ValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !h.checkHash(r) {
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
		v, err := h.storage.GetGauge(m.ID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Value = &v
	case "counter":
		v, err := h.storage.GetCounter(m.ID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Delta = &v
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

	responseData, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Ошибка при кодировании JSON ответа в ValueJSONHandler: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.addHashToResponse(w, responseData)

	if _, err := w.Write(responseData); err != nil {
		log.Printf("Ошибка при записи ответа в ValueJSONHandler: %v", err)
	}
}

// PingHandler обрабатывает GET запросы для проверки соединения с базой данных
func (h *Handlers) PingHandler(w http.ResponseWriter, r *http.Request) {
	if !h.storage.IsAvailable() {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.storage.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// UpdatesHandler обрабатывает POST запросы для обновления множества метрик в JSON формате
func (h *Handlers) UpdatesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !h.checkHash(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var metrics []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Обновляем все метрики в батче одной операцией
	if err := h.storage.UpdateBatch(metrics); err != nil {
		log.Printf("Failed to update batch: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
