package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"compress/gzip"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	flagRunAddr    string
	reportInterval time.Duration
	pollInterval   time.Duration
	key            string
	rateLimit      int
)

func parseFlags() error {
	var a string
	var r, p int
	var k string
	var l int

	flag.StringVar(&a, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&r, "r", 10, "report interval in seconds")
	flag.IntVar(&p, "p", 2, "poll interval in seconds")
	flag.StringVar(&k, "k", "", "signature key")
	flag.IntVar(&l, "l", 1, "rate limit for concurrent requests")
	flag.Parse()

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		a = envRunAddr
	}
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		ri, err := strconv.Atoi(envReportInterval)
		if err != nil || ri <= 0 {
			return fmt.Errorf("сonfiguration error: incorrect value REPORT_INTERVAL: %v", envReportInterval)
		}
		r = ri
	}
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		pi, err := strconv.Atoi(envPollInterval)
		if err != nil || pi <= 0 {
			return fmt.Errorf("сonfiguration error: incorrect value POLL_INTERVAL: %v", envPollInterval)
		}
		p = pi
	}
	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		rl, err := strconv.Atoi(envRateLimit)
		if err != nil || rl <= 0 {
			return fmt.Errorf("сonfiguration error: incorrect value RATE_LIMIT: %v", envRateLimit)
		}
		l = rl
	}

	if r <= 0 {
		return fmt.Errorf("сonfiguration error: REPORT_INTERVAL должен быть больше 0")
	}
	if p <= 0 {
		return fmt.Errorf("сonfiguration error: POLL_INTERVAL должен быть больше 0")
	}
	if l <= 0 {
		return fmt.Errorf("сonfiguration error: RATE_LIMIT должен быть больше 0")
	}

	if envKey := os.Getenv("KEY"); envKey != "" {
		k = envKey
	}

	flagRunAddr = fmt.Sprintf("http://%s", a)
	reportInterval = time.Duration(r) * time.Second
	pollInterval = time.Duration(p) * time.Second
	key = k
	rateLimit = l
	return nil
}

// Интерфейс для сборщика метрик
//
//go:generate mockgen -destination=mocks/collector.go -package=mocks . Collector
type Collector interface {
	Start()
	Stop()
	Metrics() <-chan []models.Metrics
}

// Интерфейс для отправителя метрик
//
//go:generate mockgen -destination=mocks/sender.go -package=mocks . Sender
type Sender interface {
	Start()
	Stop()
	Metrics() chan<- []models.Metrics
}

var _ Collector = (*MetricsCollector)(nil)
var _ Sender = (*MetricsSender)(nil)

// MetricsCollector собирает метрики
type MetricsCollector struct {
	metricsChan chan []models.Metrics
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewMetricsCollector создает новый коллектор метрик
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metricsChan: make(chan []models.Metrics, 100),
		stopChan:    make(chan struct{}),
	}
}

// Start запускает сбор метрик
func (mc *MetricsCollector) Start() {
	mc.wg.Add(1)
	go mc.collectRuntimeMetrics()

	mc.wg.Add(1)
	go mc.collectSystemMetrics()
}

// Stop останавливает сбор метрик
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
	mc.wg.Wait()
	close(mc.metricsChan)
}

// Metrics возвращает канал с метриками
func (mc *MetricsCollector) Metrics() <-chan []models.Metrics {
	return mc.metricsChan
}

// NewMetric создаёт метрики с хешем
func NewMetric(id, mType string, value *float64, delta *int64, key string) models.Metrics {
	metric := models.Metrics{
		ID:    id,
		MType: mType,
		Value: value,
		Delta: delta,
	}
	if key != "" {
		var data string
		if mType == "gauge" && value != nil {
			data = fmt.Sprintf("%s:%s:%f", id, mType, *value)
		} else if mType == "counter" && delta != nil {
			data = fmt.Sprintf("%s:%s:%d", id, mType, *delta)
		}
		metric.Hash = utils.CalculateHash([]byte(data), key)
	}
	return metric
}

// collectRuntimeMetrics собирает runtime метрики
func (mc *MetricsCollector) collectRuntimeMetrics() {
	defer mc.wg.Done()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var pollCount int64

	for {
		select {
		case <-mc.stopChan:
			return
		case <-ticker.C:
			metrics := mc.collectRuntimeMetricsData()

			// Добавляем счетчик опросов
			pollCount++
			pc := pollCount
			metrics = append(metrics, NewMetric("PollCount", "counter", nil, &pc, key))

			select {
			case mc.metricsChan <- metrics:
			case <-mc.stopChan:
				return
			}
		}
	}
}

// collectSystemMetrics собирает системные метрики через gopsutil
func (mc *MetricsCollector) collectSystemMetrics() {
	defer mc.wg.Done()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.stopChan:
			return
		case <-ticker.C:
			metrics := mc.collectSystemMetricsData()

			select {
			case mc.metricsChan <- metrics:
			case <-mc.stopChan:
				return
			}
		}
	}
}

// collectRuntimeMetricsData собирает runtime метрики
func (mc *MetricsCollector) collectRuntimeMetricsData() []models.Metrics {
	var metrics []models.Metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	gaugeMetrics := map[string]float64{
		"Alloc":         float64(m.Alloc),
		"BuckHashSys":   float64(m.BuckHashSys),
		"Frees":         float64(m.Frees),
		"GCCPUFraction": m.GCCPUFraction,
		"GCSys":         float64(m.GCSys),
		"HeapAlloc":     float64(m.HeapAlloc),
		"HeapIdle":      float64(m.HeapIdle),
		"HeapInuse":     float64(m.HeapInuse),
		"HeapObjects":   float64(m.HeapObjects),
		"HeapReleased":  float64(m.HeapReleased),
		"HeapSys":       float64(m.HeapSys),
		"LastGC":        float64(m.LastGC),
		"Lookups":       float64(m.Lookups),
		"MCacheInuse":   float64(m.MCacheInuse),
		"MCacheSys":     float64(m.MCacheSys),
		"MSpanInuse":    float64(m.MSpanInuse),
		"MSpanSys":      float64(m.MSpanSys),
		"Mallocs":       float64(m.Mallocs),
		"NextGC":        float64(m.NextGC),
		"NumForcedGC":   float64(m.NumForcedGC),
		"NumGC":         float64(m.NumGC),
		"OtherSys":      float64(m.OtherSys),
		"PauseTotalNs":  float64(m.PauseTotalNs),
		"StackInuse":    float64(m.StackInuse),
		"StackSys":      float64(m.StackSys),
		"Sys":           float64(m.Sys),
		"TotalAlloc":    float64(m.TotalAlloc),
	}

	for name, value := range gaugeMetrics {
		v := value
		metrics = append(metrics, NewMetric(name, "gauge", &v, nil, key))
	}

	// Добавляем случайное значение
	rv := rand.Float64()
	metrics = append(metrics, NewMetric("RandomValue", "gauge", &rv, nil, key))

	return metrics
}

// collectSystemMetricsData собирает системные метрики через gopsutil
func (mc *MetricsCollector) collectSystemMetricsData() []models.Metrics {
	var metrics []models.Metrics

	// Собираем метрики памяти
	if vmstat, err := mem.VirtualMemory(); err == nil {
		totalMemory := float64(vmstat.Total)
		metrics = append(metrics, NewMetric("TotalMemory", "gauge", &totalMemory, nil, key))

		freeMemory := float64(vmstat.Free)
		metrics = append(metrics, NewMetric("FreeMemory", "gauge", &freeMemory, nil, key))
	}

	// Собираем метрики CPU
	if cpuPercentages, err := cpu.Percent(0, true); err == nil {
		for i, percentage := range cpuPercentages {
			metrics = append(metrics, NewMetric(fmt.Sprintf("CPUutilization%d", i+1), "gauge", &percentage, nil, key))
		}
	}

	return metrics
}

// MetricsSender отправляет метрики на сервер
type MetricsSender struct {
	metricsChan chan []models.Metrics
	stopChan    chan struct{}
	wg          sync.WaitGroup
	semaphore   chan struct{} // Семафор для ограничения количества запросов
}

// NewMetricsSender создает новый отправитель метрик
func NewMetricsSender() *MetricsSender {
	return &MetricsSender{
		metricsChan: make(chan []models.Metrics, 100),
		stopChan:    make(chan struct{}),
		semaphore:   make(chan struct{}, rateLimit), // Worker pool с ограничением
	}
}

// Start запускает отправку метрик
func (ms *MetricsSender) Start() {
	// Запускаем worker pool
	for i := 0; i < rateLimit; i++ {
		ms.wg.Add(1)
		go ms.worker()
	}
}

// Stop останавливает отправку метрик
func (ms *MetricsSender) Stop() {
	close(ms.stopChan)
	ms.wg.Wait()
	close(ms.metricsChan)
}

// Metrics возвращает канал для отправки метрик
func (ms *MetricsSender) Metrics() chan<- []models.Metrics {
	return ms.metricsChan
}

// worker - воркер для отправки метрик
func (ms *MetricsSender) worker() {
	defer ms.wg.Done()

	for {
		select {
		case <-ms.stopChan:
			return
		case metrics := <-ms.metricsChan:
			// Получаем слот в семафоре
			ms.semaphore <- struct{}{}

			// Отправляем метрики
			if err := ms.sendMetricsBatch(metrics); err != nil {
				log.Printf("Error sending metrics batch: %v", err)
			}

			// Освобождаем слот
			<-ms.semaphore
		}
	}
}

// sendMetricsBatch отправляет множество метрик одним запросом
func (ms *MetricsSender) sendMetricsBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	url, err := url.JoinPath(flagRunAddr, "updates/")
	if err != nil {
		return fmt.Errorf("error joining URL: %w", err)
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(body)
	if err != nil {
		return fmt.Errorf("gzip write error: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return utils.Retry(ctx, utils.DefaultRetryConfig(), func() error {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error sending batch request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return nil
	})
}

func main() {
	if err := parseFlags(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting agent with rate limit: %d", rateLimit)

	var collector Collector = NewMetricsCollector()
	var sender Sender = NewMetricsSender()

	collector.Start()
	sender.Start()

	go func() {
		for metrics := range collector.Metrics() {
			select {
			case sender.Metrics() <- metrics:
			case <-collector.(*MetricsCollector).stopChan:
				return
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down agent...")
	collector.Stop()
	sender.Stop()
	log.Println("Agent stopped")
}
