package app

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
)

// TestNewApp тестирует создание нового приложения.
func TestNewApp(t *testing.T) {
	collector := NewMetricsCollector(&AgentConfig{PollInterval: 2, Key: "test"})
	sender := NewMetricsSender(&AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"})

	app := NewApp(collector, sender)
	if app == nil {
		t.Fatal("NewApp не должен возвращать nil")
	}

	if app.collector != collector {
		t.Error("Collector должен быть установлен корректно")
	}

	if app.sender != sender {
		t.Error("Sender должен быть установлен корректно")
	}
}

// TestNewMetric тестирует создание новой метрики.
func TestNewMetric(t *testing.T) {
	value := 123.45
	metric := NewMetric("test", "gauge", &value, nil, "test-key")

	if metric.ID != "test" {
		t.Errorf("Ожидался ID 'test', получен '%s'", metric.ID)
	}

	if metric.MType != "gauge" {
		t.Errorf("Ожидался тип 'gauge', получен '%s'", metric.MType)
	}

	if metric.Value == nil || *metric.Value != 123.45 {
		t.Error("Значение метрики должно быть установлено корректно")
	}
}

// TestNewMetricsCollector тестирует создание сборщика метрик.
func TestNewMetricsCollector(t *testing.T) {
	cfg := &AgentConfig{PollInterval: 2, Key: "test"}
	collector := NewMetricsCollector(cfg)
	if collector == nil {
		t.Fatal("NewMetricsCollector не должен возвращать nil")
	}
}

// TestNewWorkerPool тестирует создание пула воркеров.
func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool(5)
	if pool == nil {
		t.Fatal("NewWorkerPool не должен возвращать nil")
	}
}

// TestNewMetricsSender тестирует создание отправителя метрик.
func TestNewMetricsSender(t *testing.T) {
	cfg := &AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"}
	sender := NewMetricsSender(cfg)
	if sender == nil {
		t.Fatal("NewMetricsSender не должен возвращать nil")
	}
}

// TestWorkerPoolLifecycle тестирует жизненный цикл пула воркеров.
func TestWorkerPoolLifecycle(t *testing.T) {
	pool := NewWorkerPool(2)
	ctx, cancel := context.WithCancel(context.Background())

	pool.Start(ctx)

	pool.Submit(func() {
		time.Sleep(10 * time.Millisecond)
	})

	cancel()
	pool.Stop()
}

// TestMetricsSenderLifecycle тестирует жизненный цикл отправителя метрик.
func TestMetricsSenderLifecycle(t *testing.T) {
	cfg := &AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"}
	sender := NewMetricsSender(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	sender.Start(ctx)

	cancel()
	sender.Stop()
}

// TestAppRun тестирует запуск приложения.
func TestAppRun(t *testing.T) {
	collector := NewMetricsCollector(&AgentConfig{PollInterval: 2, Key: "test"})
	sender := NewMetricsSender(&AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"})

	app := NewApp(collector, sender)
	ctx, cancel := context.WithCancel(context.Background())

	// Запускаем приложение в горутине
	go func() {
		app.Run(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Завершаем приложение через cancel
	cancel()

	time.Sleep(10 * time.Millisecond)
}

// TestAppStop тестирует остановку приложения.
func TestAppStop(t *testing.T) {
	collector := NewMetricsCollector(&AgentConfig{PollInterval: 2, Key: "test"})
	sender := NewMetricsSender(&AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"})

	app := NewApp(collector, sender)

	// Останавливаем приложение без запуска
	app.collector.Stop()
	app.sender.Stop()
}

// TestParseAgentConfig тестирует парсинг конфигурации агента.
func TestParseAgentConfig(t *testing.T) {
	cfg, err := ParseAgentConfig()
	if err != nil {
		t.Fatalf("ParseAgentConfig() error: %v", err)
	}

	if cfg.Address != "localhost:8080" {
		t.Errorf("Ожидался адрес 'localhost:8080', получен '%s'", cfg.Address)
	}

	if cfg.ReportInterval != 10 {
		t.Errorf("Ожидался интервал отчетов 10, получен %d", cfg.ReportInterval)
	}

	if cfg.PollInterval != 2 {
		t.Errorf("Ожидался интервал опросов 2, получен %d", cfg.PollInterval)
	}
}

// TestWorkerPoolSubmitAfterStop тестирует отправку задачи после Stop.
func TestWorkerPoolSubmitAfterStop(t *testing.T) {
	pool := NewWorkerPool(1)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	cancel()

	time.Sleep(10 * time.Millisecond)

	pool.Stop()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Ожидалась паника при отправке задачи после Stop")
		}
	}()
	pool.Submit(func() {})
}

// TestMetricsSenderSendMetricsBatchEmpty тестирует SendMetricsBatch с пустым слайсом.
func TestMetricsSenderSendMetricsBatchEmpty(t *testing.T) {
	sender := NewMetricsSender(&AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"}).(*MetricsSender)
	err := sender.SendMetricsBatch(nil)
	if err != nil {
		t.Errorf("SendMetricsBatch(nil) не должен возвращать ошибку: %v", err)
	}
}

// TestMetricsSenderMetricsChannelClose тестирует закрытие канала метрик.
func TestMetricsSenderMetricsChannelClose(t *testing.T) {
	sender := NewMetricsSender(&AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"}).(*MetricsSender)
	sender.Stop()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Ожидалась паника при отправке в закрытый канал")
		}
	}()
	sender.Metrics() <- nil
}

// TestMetricsCollectorStartStop тестирует запуск и остановку MetricsCollector.
func TestMetricsCollectorStartStop(t *testing.T) {
	cfg := &AgentConfig{PollInterval: 1, Key: "test"}
	collector := NewMetricsCollector(cfg).(*MetricsCollector)
	ctx, cancel := context.WithCancel(context.Background())
	collector.Start(ctx)
	cancel()
	collector.Stop()
}

// TestMetricsCollectorMetricsChannel тестирует получение канала метрик.
func TestMetricsCollectorMetricsChannel(t *testing.T) {
	cfg := &AgentConfig{PollInterval: 1, Key: "test"}
	collector := NewMetricsCollector(cfg).(*MetricsCollector)
	ch := collector.Metrics()
	if ch == nil {
		t.Error("Metrics должен возвращать канал")
	}
}

// TestCollectRuntimeMetricsData тестирует сбор runtime-метрик.
func TestCollectRuntimeMetricsData(t *testing.T) {
	cfg := &AgentConfig{PollInterval: 1, Key: "test"}
	collector := NewMetricsCollector(cfg).(*MetricsCollector)
	metrics := collector.CollectRuntimeMetricsData()
	if len(metrics) == 0 {
		t.Error("CollectRuntimeMetricsData должен возвращать метрики")
	}
}

// TestCollectSystemMetricsData тестирует сбор системных метрик.
func TestCollectSystemMetricsData(t *testing.T) {
	cfg := &AgentConfig{PollInterval: 1, Key: "test"}
	collector := NewMetricsCollector(cfg).(*MetricsCollector)
	metrics := collector.CollectSystemMetricsData()
	if len(metrics) == 0 {
		t.Error("CollectSystemMetricsData должен возвращать метрики")
	}
}

// TestCollectRuntimeMetricsCancel тестирует корректное завершение collectRuntimeMetrics по ctx.Done().
func TestCollectRuntimeMetricsCancel(t *testing.T) {
	cfg := &AgentConfig{PollInterval: 1, Key: "test"}
	collector := NewMetricsCollector(cfg).(*MetricsCollector)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	collector.wg.Add(1)
	go func() {
		collector.collectRuntimeMetrics(ctx)
		close(ch)
	}()
	cancel()
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Error("collectRuntimeMetrics не завершился по ctx.Done()")
	}
}

// TestCollectSystemMetricsCancel тестирует корректное завершение collectSystemMetrics по ctx.Done().
func TestCollectSystemMetricsCancel(t *testing.T) {
	cfg := &AgentConfig{PollInterval: 1, Key: "test"}
	collector := NewMetricsCollector(cfg).(*MetricsCollector)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	collector.wg.Add(1)
	go func() {
		collector.collectSystemMetrics(ctx)
		close(ch)
	}()
	cancel()
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Error("collectSystemMetrics не завершился по ctx.Done()")
	}
}

// TestSendMetricsBatch_InvalidURL тестирует ошибку при некорректном адресе.
func TestSendMetricsBatch_InvalidURL(t *testing.T) {
	sender := NewMetricsSender(&AgentConfig{Address: ":://bad_url", RateLimit: 1, Key: "test"}).(*MetricsSender)
	metrics := []models.Metrics{{ID: "test", MType: "gauge", Value: func() *float64 { v := 1.0; return &v }()}}
	err := sender.SendMetricsBatch(metrics)
	if err == nil {
		t.Error("Ожидалась ошибка при некорректном адресе")
	}
}

// TestSendMetricsBatch_MarshalError тестирует ошибку marshal (невозможно сериализовать).
func TestSendMetricsBatch_MarshalError(t *testing.T) {
	sender := NewMetricsSender(&AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"}).(*MetricsSender)
	// Используем канал вместо слайса, чтобы вызвать ошибку marshal
	var wrongType interface{} = make(chan int)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Ожидалась паника или ошибка marshal")
		}
	}()
	_ = sender.SendMetricsBatch(wrongType.([]models.Metrics))
}

// TestSendMetricsBatch_GzipError тестирует ошибку gzip (например, закрытый буфер).
func TestSendMetricsBatch_GzipError(t *testing.T) {
	sender := NewMetricsSender(&AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: "test"}).(*MetricsSender)
	metrics := []models.Metrics{{ID: "test", MType: "gauge", Value: func() *float64 { v := 1.0; return &v }()}}
	oldBufPool := bufPool
	bufPool = sync.Pool{New: func() interface{} { return nil }}
	defer func() { bufPool = oldBufPool }()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Ожидалась паника при ошибке gzip")
		}
	}()
	_ = sender.SendMetricsBatch(metrics)
}
