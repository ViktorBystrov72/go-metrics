package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectRuntimeMetricsData(t *testing.T) {
	collector := NewMetricsCollector()
	metrics := collector.collectRuntimeMetricsData()

	metricNames := make(map[string]bool)
	for _, m := range metrics {
		metricNames[m.ID] = true
	}

	requiredMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	for _, name := range requiredMetrics {
		assert.True(t, metricNames[name], "Metric %s is missing", name)
	}

	for _, m := range metrics {
		assert.Equal(t, "gauge", m.MType)
	}
}

func TestCollectSystemMetricsData(t *testing.T) {
	collector := NewMetricsCollector()
	metrics := collector.collectSystemMetricsData()

	metricNames := make(map[string]bool)
	for _, m := range metrics {
		metricNames[m.ID] = true
	}

	// Проверяем наличие системных метрик
	assert.True(t, metricNames["TotalMemory"], "TotalMemory metric is missing")
	assert.True(t, metricNames["FreeMemory"], "FreeMemory metric is missing")

	// Проверяем наличие хотя бы одной метрики CPU
	hasCPU := false
	for name := range metricNames {
		if len(name) >= 14 && name[:14] == "CPUutilization" {
			hasCPU = true
			break
		}
	}
	assert.True(t, hasCPU, "CPU utilization metrics are missing")

	for _, m := range metrics {
		assert.Equal(t, "gauge", m.MType)
	}
}

func TestSendMetricsBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	flagRunAddr = server.URL

	sender := NewMetricsSender()
	val1 := 123.45
	val2 := 67.89
	metrics := []models.Metrics{
		{
			ID:    "testMetric1",
			MType: "gauge",
			Value: &val1,
		},
		{
			ID:    "testMetric2",
			MType: "gauge",
			Value: &val2,
		},
	}

	err := sender.sendMetricsBatch(metrics)
	require.NoError(t, err)
}

func TestMetricsCollector(t *testing.T) {
	pollInterval = 1 * time.Second
	reportInterval = 2 * time.Second

	collector := NewMetricsCollector()

	collector.Start()

	time.Sleep(100 * time.Millisecond)

	collector.Stop()

	_, ok := <-collector.Metrics()
	assert.False(t, ok, "Metrics channel should be closed")
}

func TestMetricsSender(t *testing.T) {
	rateLimit = 1

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	flagRunAddr = server.URL

	sender := NewMetricsSender()

	sender.Start()

	val := 123.45
	metrics := []models.Metrics{
		{
			ID:    "testMetric",
			MType: "gauge",
			Value: &val,
		},
	}

	sender.Metrics() <- metrics

	time.Sleep(100 * time.Millisecond)

	sender.Stop()

}
