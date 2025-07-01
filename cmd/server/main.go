package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/logger"
	"github.com/ViktorBystrov72/go-metrics/internal/middleware"
	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Server struct {
	storage storage.Storage
}

func NewServer(storage storage.Storage) *Server {
	return &Server{
		storage: storage,
	}
}

// updateHandler обрабатывает POST запросы для обновления метрик
func (s *Server) updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
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
		s.storage.UpdateGauge(name, v)
	case string(storage.Counter):
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.storage.UpdateCounter(name, v)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "OK")
}

// valueHandler обрабатывает GET запросы для получения значений метрик
func (s *Server) valueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	if name == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	switch metricType {
	case string(storage.Gauge):
		value, exists := s.storage.GetGauge(name)
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, value)
	case string(storage.Counter):
		value, exists := s.storage.GetCounter(name)
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, value)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// indexHandler обрабатывает GET запросы для отображения HTML-страницы со всеми метриками
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	gauges := s.storage.GetAllGauges()
	counters := s.storage.GetAllCounters()

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

func (s *Server) updateJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
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
		s.storage.UpdateGauge(m.ID, *m.Value)
		v, _ := s.storage.GetGauge(m.ID)
		resp.Value = &v
	case "counter":
		if m.Delta == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.storage.UpdateCounter(m.ID, *m.Delta)
		v, _ := s.storage.GetCounter(m.ID)
		resp.Delta = &v
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) valueJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
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
		v, ok := s.storage.GetGauge(m.ID)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Value = &v
	case "counter":
		v, ok := s.storage.GetCounter(m.ID)
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
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	// Конфигурация
	var (
		flagRunAddr         string
		flagStoreInterval   int
		flagFileStoragePath string
		flagRestore         bool
	)
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&flagStoreInterval, "i", 300, "store interval in seconds")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&flagRestore, "r", true, "restore from file on start")
	flag.Parse()

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		if v, err := strconv.Atoi(envStoreInterval); err == nil {
			flagStoreInterval = v
		}
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		flagFileStoragePath = envFileStoragePath
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if envRestore == "true" || envRestore == "1" {
			flagRestore = true
		} else if envRestore == "false" || envRestore == "0" {
			flagRestore = false
		}
	}

	storage := storage.NewMemStorage()
	if flagRestore {
		_ = storage.LoadFromFile(flagFileStoragePath) // игнорируем ошибку если файла нет
	}
	server := NewServer(storage)

	r := chi.NewRouter()

	r.Use(middleware.GzipMiddleware)

	// Маршруты для обновления метрик
	r.Route("/update", func(r chi.Router) {
		r.Post("/{type}/{name}/{value}", server.updateHandler)
	})

	// Маршруты для получения значений метрик
	r.Route("/value", func(r chi.Router) {
		r.Get("/{type}/{name}", server.valueHandler)
	})

	// Главная страница со списком всех метрик
	r.Get("/", server.indexHandler)

	// JSON API
	r.Post("/update/", server.updateJSONHandler)
	r.Post("/value/", server.valueJSONHandler)

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("cannot initialize zap logger: %v", err)
	}
	defer zapLogger.Sync()

	loggedRouter := logger.WithLogging(zapLogger, r)

	// Периодическое сохранение
	stop := make(chan struct{})
	if flagStoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(flagStoreInterval) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_ = storage.SaveToFile(flagFileStoragePath)
				case <-stop:
					return
				}
			}
		}()
	}

	// Сохранение при завершении
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		_ = storage.SaveToFile(flagFileStoragePath)
		os.Exit(0)
	}()

	log.Fatal(http.ListenAndServe(flagRunAddr, loggedRouter))
}
