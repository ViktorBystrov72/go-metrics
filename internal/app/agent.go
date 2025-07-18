package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/crypto"
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

	// WaitGroup для отслеживания горутины переправки метрик
	var transferWg sync.WaitGroup
	transferWg.Add(1)

	go func() {
		defer transferWg.Done()
		for metrics := range a.collector.Metrics() {
			select {
			case a.sender.Metrics() <- metrics:
			case <-ctx.Done():
				return
			}
		}
		log.Printf("Завершена передача метрик из collector в sender")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigChan
	log.Printf("Получен сигнал %v, выполняем graceful shutdown...", sig)

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Канал для сигнализации о завершении shutdown
	shutdownDone := make(chan struct{})

	go func() {
		defer close(shutdownDone)

		// 1. Сначала останавливаем сбор метрик
		log.Printf("Остановка сбора метрик...")
		a.collector.Stop()

		// 2. Ждем завершения передачи всех собранных метрик
		log.Printf("Ожидание завершения передачи метрик...")
		transferWg.Wait()

		// 3. Останавливаем отправителя
		log.Printf("Остановка отправки метрик...")
		a.sender.Stop()
	}()

	// Ждем либо завершения shutdown, либо таймаута
	select {
	case <-shutdownDone:
		log.Println("Agent stopped gracefully")
	case <-shutdownCtx.Done():
		log.Printf("Graceful shutdown timeout exceeded, forcing shutdown...")
		// При таймауте принудительно завершаем
		// Компоненты должны реагировать на отмену контекста
	}

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

	// Поля для graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMetricsCollector создаёт новый Collector с учётом конфига
func NewMetricsCollector(cfg *AgentConfig) Collector {
	ctx, cancel := context.WithCancel(context.Background())
	return &MetricsCollector{
		metricsChan:  make(chan []models.Metrics, 100),
		pollInterval: time.Duration(cfg.PollInterval) * time.Second,
		key:          cfg.Key,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start запускает сбор метрик
func (mc *MetricsCollector) Start(ctx context.Context) {
	mc.wg.Add(1)
	go mc.collectRuntimeMetrics(mc.ctx)

	mc.wg.Add(1)
	go mc.collectSystemMetrics(mc.ctx)
}

// Stop останавливает сбор метрик
func (mc *MetricsCollector) Stop() {
	mc.cancel()

	// Ждем завершения горутин с таймаутом
	done := make(chan struct{})
	go func() {
		mc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		log.Printf("MetricsCollector.Stop() timeout exceeded, some goroutines may not have finished")
	}

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
	// Ждем завершения воркеров с таймаутом
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Printf("WorkerPool.Stop() timeout exceeded, some workers may not have finished")
	}

	close(wp.tasks)
}

// MetricsSender формирует задачи отправки и кладёт их в pool
type MetricsSender struct {
	metricsChan chan []models.Metrics
	pool        Pool
	address     string
	key         string
	publicKey   *rsa.PublicKey

	// Поля для graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMetricsSender создаёт новый Sender с учётом конфига
func NewMetricsSender(cfg *AgentConfig) Sender {
	var publicKey *rsa.PublicKey

	// Загружаем публичный ключ, если путь указан
	if cfg.CryptoKey != "" {
		var err error
		publicKey, err = crypto.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			log.Printf("Ошибка загрузки публичного ключа: %v", err)
			// Продолжаем работу без шифрования
		} else {
			log.Printf("Публичный ключ загружен из: %s", cfg.CryptoKey)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &MetricsSender{
		metricsChan: make(chan []models.Metrics, 100),
		pool:        NewWorkerPool(cfg.RateLimit),
		address:     fmt.Sprintf("http://%s", cfg.Address),
		key:         cfg.Key,
		publicKey:   publicKey,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (ms *MetricsSender) Start(ctx context.Context) {
	ms.pool.Start(ms.ctx)
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
	// Отменяем контекст, чтобы остановить WorkerPool
	ms.cancel()

	done := make(chan struct{})
	go func() {
		ms.pool.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		log.Printf("MetricsSender.Stop() timeout exceeded, WorkerPool may not have stopped completely")
	}

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

// getHostIP возвращает IP-адрес хоста для заголовка X-Real-IP
func getHostIP() string {
	// Пытаемся подключиться к внешнему адресу для определения локального IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
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

	// Сжимаем данные
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

	// Получаем сжатые данные
	compressedData := buf.Bytes()
	var finalData []byte
	var contentEncoding string

	// Шифруем данные, если есть публичный ключ
	if ms.publicKey != nil {
		encryptedData, err := crypto.EncryptLargeData(compressedData, ms.publicKey)
		if err != nil {
			return fmt.Errorf("encryption error: %w", err)
		}

		// Кодируем в Base64 для передачи
		finalData = []byte(base64.StdEncoding.EncodeToString(encryptedData))
		contentEncoding = "encrypted"
	} else {
		finalData = compressedData
		contentEncoding = "gzip"
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(finalData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", contentEncoding)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Real-IP", getHostIP())

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
