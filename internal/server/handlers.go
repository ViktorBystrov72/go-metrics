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

// addHashToMetrics добавляет хеш к метрике
func (h *Handlers) addHashToMetrics(m *models.Metrics) {
	if h.key == "" {
		return
	}

	var data string
	switch m.MType {
	case "counter":
		if m.Delta != nil {
			data = fmt.Sprintf("%s:%s:%d", m.ID, m.MType, *m.Delta)
		}
	case "gauge":
		if m.Value != nil {
			data = fmt.Sprintf("%s:%s:%f", m.ID, m.MType, *m.Value)
		}
	}

	if data != "" {
		m.Hash = utils.CalculateHash([]byte(data), h.key)
	}
}

// checkHash проверяет хеш запроса
func (h *Handlers) checkHash(r *http.Request) bool {
	if h.key == "" {
		return true // если ключ не задан, проверка не требуется
	}

	// Для JSON запросов проверяем хеш из тела запроса
	if r.Header.Get("Content-Type") == "application/json" {
		return h.checkJSONHash(r)
	}

	// Для обычных запросов проверяем хеш из заголовков
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}

	// Восстанавливаем тело запроса для дальнейшего использования
	r.Body = io.NopCloser(bytes.NewReader(body))

	receivedHash := r.Header.Get("HashSHA256")
	if receivedHash == "" {
		// Поддерживаем также заголовок Hash для обратной совместимости
		receivedHash = r.Header.Get("Hash")
		if receivedHash == "" {
			return true // если хеш не передан, пропускаем проверку
		}
		if receivedHash == "none" {
			return true // специальное значение означает пропуск проверки хеша
		}
	}

	return utils.VerifyHash(body, h.key, receivedHash)
}

// checkJSONHash проверяет хеш для JSON запросов
func (h *Handlers) checkJSONHash(r *http.Request) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}

	// Восстанавливаем тело запроса для дальнейшего использования
	r.Body = io.NopCloser(bytes.NewReader(body))

	headerHash := r.Header.Get("HashSHA256")
	if headerHash == "" {
		headerHash = r.Header.Get("Hash")
	}

	// Выход для специального значения "none"
	if headerHash == "none" {
		return true
	}

	// Парсим JSON чтобы получить хеш
	var m models.Metrics
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}

	// Выход, если хеш не передан ни в заголовке, ни в теле
	if m.Hash == "" && headerHash == "" {
		return true
	}

	// Выход для проверки хеша из заголовка
	if headerHash != "" {
		return utils.VerifyHash(body, h.key, headerHash)
	}

	// Выход, если хеш не передан в JSON теле
	if m.Hash == "" {
		return false
	}

	// Вычисляем ожидаемый хеш для JSON тела
	var data string
	switch m.MType {
	case "counter":
		if m.Delta == nil {
			return false
		}
		data = fmt.Sprintf("%s:%s:%d", m.ID, m.MType, *m.Delta)
	case "gauge":
		if m.Value == nil {
			return false
		}
		data = fmt.Sprintf("%s:%s:%f", m.ID, m.MType, *m.Value)
	default:
		return false
	}

	return utils.VerifyHash([]byte(data), h.key, m.Hash)
}

// verifyMetricHash проверяет хеш отдельной метрики
func (h *Handlers) verifyMetricHash(m models.Metrics) bool {
	// Если ключ не задан
	if h.key == "" {
		return true
	}

	// Если хеш не передан
	if m.Hash == "" {
		log.Printf("No hash provided for metric: %s, type: %s", m.ID, m.MType)
		return false
	}

	// Вычисляем ожидаемый хеш
	var data string
	switch m.MType {
	case "counter":
		if m.Delta == nil {
			log.Printf("Counter metric %s has nil delta", m.ID)
			return false
		}
		data = fmt.Sprintf("%s:%s:%d", m.ID, m.MType, *m.Delta)
	case "gauge":
		if m.Value == nil {
			log.Printf("Gauge metric %s has nil value", m.ID)
			return false
		}
		data = fmt.Sprintf("%s:%s:%f", m.ID, m.MType, *m.Value)
	default:
		log.Printf("Unknown metric type: %s for metric %s", m.MType, m.ID)
		return false
	}

	expectedHash := utils.CalculateHash([]byte(data), h.key)
	if m.Hash != expectedHash {
		log.Printf("Hash mismatch for metric %s: expected %s, got %s", m.ID, expectedHash, m.Hash)
		return false
	}

	return true
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

	// Добавляем хеш к ответу
	h.addHashToMetrics(&resp)

	w.WriteHeader(http.StatusOK)

	responseData, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Ошибка при кодировании JSON ответа в UpdateJSONHandler: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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

	// Добавляем хеш к ответу
	h.addHashToMetrics(&resp)

	w.WriteHeader(http.StatusOK)

	responseData, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Ошибка при кодировании JSON ответа в ValueJSONHandler: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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

	// Для batch запросов не проверяем хеш, так как это массив метрик
	// и каждая метрика может иметь свой хеш

	var metrics []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Проверяем хеш каждой метрики если ключ задан
	if h.key != "" {
		for i, metric := range metrics {
			if !h.verifyMetricHash(metric) {
				log.Printf("Hash verification failed for metric %d: %s, type: %s", i, metric.ID, metric.MType)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
	}

	// Группируем метрики по ключу (name, type) для избежания дубликатов в одном батче
	metricsMap := make(map[string]models.Metrics)
	for _, metric := range metrics {
		key := metric.ID + "_" + metric.MType
		if existing, exists := metricsMap[key]; exists {
			// Если метрика уже есть, объединяем значения
			if metric.MType == "counter" && metric.Delta != nil && existing.Delta != nil {
				combinedDelta := *existing.Delta + *metric.Delta
				metric.Delta = &combinedDelta
			}
		}
		metricsMap[key] = metric
	}

	uniqueMetrics := make([]models.Metrics, 0, len(metricsMap))
	for _, metric := range metricsMap {
		uniqueMetrics = append(uniqueMetrics, metric)
	}

	// Обновляем все метрики в батче одной операцией
	if err := h.storage.UpdateBatch(uniqueMetrics); err != nil {
		log.Printf("Failed to update batch: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
