package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

var baseURL = "http://localhost:8080"

type Metric struct {
	Type  string
	Name  string
	Value string
}

func collectMetrics() []Metric {
	var metrics []Metric
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
		metrics = append(metrics, Metric{
			Type:  "gauge",
			Name:  name,
			Value: fmt.Sprintf("%g", value),
		})
	}

	metrics = append(metrics, Metric{
		Type:  "gauge",
		Name:  "RandomValue",
		Value: fmt.Sprintf("%g", rand.Float64()),
	})

	return metrics
}

func sendMetric(metric Metric) error {
	url := fmt.Sprintf("%s/update/%s/%s/%s",
		baseURL, metric.Type, metric.Name, metric.Value)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	pollInterval := 2 * time.Second
	var pollCount int64

	for {
		metrics := collectMetrics()
		pollCount++

		metrics = append(metrics, Metric{
			Type:  "counter",
			Name:  "PollCount",
			Value: fmt.Sprintf("%d", pollCount),
		})

		for _, metric := range metrics {
			if err := sendMetric(metric); err != nil {
				log.Printf("Error sending metric %s: %v", metric.Name, err)
			}
		}

		time.Sleep(pollInterval)
	}
}
