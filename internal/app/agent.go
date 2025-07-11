package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type App struct {
	collector Collector
	sender    Sender
}

func NewApp(collector Collector, sender Sender) *App {
	return &App{
		collector: collector,
		sender:    sender,
	}
}

func (a *App) Run(ctx context.Context) error {
	a.collector.Start(ctx)
	a.sender.Start(ctx)

	go func() {
		for metrics := range a.collector.Metrics() {
			select {
			case a.sender.Metrics() <- metrics:
			case <-ctx.Done():
				return
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down agent...")
	a.collector.Stop()
	a.sender.Stop()
	log.Println("Agent stopped")
	return nil
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

// MetricsCollector собирает метрики
type MetricsCollector struct {
	metricsChan  chan []models.Metrics
	wg           sync.WaitGroup
	pollInterval time.Duration
	key          string
}

// NewMetricsCollector создаёт новый Collector с учётом конфига
func NewMetricsCollector(cfg *AgentConfig) Collector {
	return &MetricsCollector{
		metricsChan:  make(chan []models.Metrics, 100),
		pollInterval: time.Duration(cfg.PollInterval) * time.Second,
		key:          cfg.Key,
	}
}

// Start запускает сбор метрик
func (mc *MetricsCollector) Start(ctx context.Context) {
	mc.wg.Add(1)
	go mc.collectRuntimeMetrics(ctx)

	mc.wg.Add(1)
	go mc.collectSystemMetrics(ctx)
}

// Stop останавливает сбор метрик
func (mc *MetricsCollector) Stop() {
	mc.wg.Wait()
	close(mc.metricsChan)
}

// Metrics возвращает канал с метриками
func (mc *MetricsCollector) Metrics() <-chan []models.Metrics {
	return mc.metricsChan
}

// collectRuntimeMetrics собирает runtime метрики
func (mc *MetricsCollector) collectRuntimeMetrics(ctx context.Context) {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.pollInterval)
	defer ticker.Stop()

	var pollCount int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := mc.CollectRuntimeMetricsData()

			// Добавляем счетчик опросов
			pollCount++
			pc := pollCount
			metrics = append(metrics, NewMetric("PollCount", "counter", nil, &pc, mc.key))

			select {
			case mc.metricsChan <- metrics:
			case <-ctx.Done():
				return
			}
		}
	}
}

// collectSystemMetrics собирает системные метрики через gopsutil
func (mc *MetricsCollector) collectSystemMetrics(ctx context.Context) {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := mc.CollectSystemMetricsData()

			select {
			case mc.metricsChan <- metrics:
			case <-ctx.Done():
				return
			}
		}
	}
}

// CollectRuntimeMetricsData собирает runtime метрики
func (mc *MetricsCollector) CollectRuntimeMetricsData() []models.Metrics {
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
		metrics = append(metrics, NewMetric(name, "gauge", &v, nil, mc.key))
	}

	// Добавляем случайное значение
	rv := rand.Float64()
	metrics = append(metrics, NewMetric("RandomValue", "gauge", &rv, nil, mc.key))

	return metrics
}

// CollectSystemMetricsData собирает системные метрики через gopsutil
func (mc *MetricsCollector) CollectSystemMetricsData() []models.Metrics {
	var metrics []models.Metrics

	// Собираем метрики памяти
	if vmstat, err := mem.VirtualMemory(); err == nil {
		totalMemory := float64(vmstat.Total)
		metrics = append(metrics, NewMetric("TotalMemory", "gauge", &totalMemory, nil, mc.key))

		freeMemory := float64(vmstat.Free)
		metrics = append(metrics, NewMetric("FreeMemory", "gauge", &freeMemory, nil, mc.key))
	}

	// Собираем метрики CPU
	if cpuPercentages, err := cpu.Percent(0, true); err == nil {
		for i, percentage := range cpuPercentages {
			metrics = append(metrics, NewMetric(fmt.Sprintf("CPUutilization%d", i+1), "gauge", &percentage, nil, mc.key))
		}
	}

	return metrics
}

type Task func()

// Интерфейс пула воркеров
type Pool interface {
	Start(ctx context.Context)
	Submit(task Task)
	Stop()
}

// WorkerPool управляет пулом воркеров для выполнения задач
type WorkerPool struct {
	tasks    chan Task
	wg       sync.WaitGroup
	poolSize int
}

func NewWorkerPool(poolSize int) *WorkerPool {
	return &WorkerPool{
		tasks:    make(chan Task, 100),
		poolSize: poolSize,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.poolSize; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task := <-wp.tasks:
					task()
				}
			}
		}()
	}
}

func (wp *WorkerPool) Submit(task Task) {
	wp.tasks <- task
}

func (wp *WorkerPool) Stop() {
	wp.wg.Wait()
	close(wp.tasks)
}

// MetricsSender формирует задачи отправки и кладёт их в pool
type MetricsSender struct {
	metricsChan chan []models.Metrics
	pool        Pool
	address     string
	key         string
}

// NewMetricsSender создаёт новый Sender с учётом конфига
func NewMetricsSender(cfg *AgentConfig) Sender {
	return &MetricsSender{
		metricsChan: make(chan []models.Metrics, 100),
		pool:        NewWorkerPool(cfg.RateLimit),
		address:     fmt.Sprintf("http://%s", cfg.Address),
		key:         cfg.Key,
	}
}

func (ms *MetricsSender) Start(ctx context.Context) {
	ms.pool.Start(ctx)
	go func() {
		for metrics := range ms.metricsChan {
			m := metrics
			ms.pool.Submit(func() {
				_ = ms.SendMetricsBatch(m)
			})
		}
	}()
}

func (ms *MetricsSender) Stop() {
	ms.pool.Stop()
	close(ms.metricsChan)
}

func (ms *MetricsSender) Metrics() chan<- []models.Metrics {
	return ms.metricsChan
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

// SendMetricsBatch отправляет множество метрик одним запросом
func (ms *MetricsSender) SendMetricsBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	url, err := url.JoinPath(ms.address, "updates/")
	if err != nil {
		return fmt.Errorf("error joining URL: %w", err)
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	gz := gzipPool.Get().(*gzip.Writer)
	gz.Reset(buf)
	defer gzipPool.Put(gz)

	_, err = gz.Write(body)
	if err != nil {
		return fmt.Errorf("gzip write error: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, buf)
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
