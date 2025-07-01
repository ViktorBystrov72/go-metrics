package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectMetrics(t *testing.T) {
	metrics := collectMetrics()

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

func TestSendMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	flagRunAddr = server.URL

	val := 123.45
	metric := models.Metrics{
		ID:    "testMetric",
		MType: "gauge",
		Value: &val,
	}

	err := sendMetric(metric)
	require.NoError(t, err)
}
