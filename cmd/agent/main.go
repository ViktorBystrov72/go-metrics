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
	"runtime"
	"strconv"
	"time"

	"compress/gzip"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
)

var (
	flagRunAddr    string
	reportInterval time.Duration
	pollInterval   time.Duration
)

func parseFlags() error {
	var a string
	var r, p int

	flag.StringVar(&a, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&r, "r", 10, "report interval in seconds")
	flag.IntVar(&p, "p", 2, "poll interval in seconds")
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

	if r <= 0 {
		return fmt.Errorf("сonfiguration error: REPORT_INTERVAL должен быть больше 0")
	}
	if p <= 0 {
		return fmt.Errorf("сonfiguration error: POLL_INTERVAL должен быть больше 0")
	}

	flagRunAddr = fmt.Sprintf("http://%s", a)
	reportInterval = time.Duration(r) * time.Second
	pollInterval = time.Duration(p) * time.Second
	return nil
}

func collectMetrics() []models.Metrics {
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
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}
	rv := rand.Float64()
	metrics = append(metrics, models.Metrics{
		ID:    "RandomValue",
		MType: "gauge",
		Value: &rv,
	})
	return metrics
}

func sendMetric(metric models.Metrics) error {
	url, err := url.JoinPath(flagRunAddr, "update/")
	if err != nil {
		return fmt.Errorf("error joining URL: %w", err)
	}
	body, err := json.Marshal(metric)
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
			return fmt.Errorf("error sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return nil
	})
}

// sendMetricsBatch отправляет множество метрик одним запросом
func sendMetricsBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil // Не отправляем пустые батчи
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
	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()
	var metricsMap = make(map[string]models.Metrics)
	var pollCount int64
	for {
		select {
		case <-pollTicker.C:
			currentMetrics := collectMetrics()
			for _, m := range currentMetrics {
				metricsMap[m.ID] = m
			}
			pollCount++
			pc := pollCount
			metricsMap["PollCount"] = models.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: &pc,
			}
		case <-reportTicker.C:
			var metricsToSend []models.Metrics
			for _, metric := range metricsMap {
				metricsToSend = append(metricsToSend, metric)
			}

			// Отправляем все метрики одним batch запросом
			if err := sendMetricsBatch(metricsToSend); err != nil {
				log.Printf("Error sending metrics batch: %v", err)
				// Fallback: отправляем по одной метрике при ошибке batch
				for _, metric := range metricsMap {
					if err := sendMetric(metric); err != nil {
						log.Printf("Error sending metric %s: %v", metric.ID, err)
					}
				}
			}

			metricsMap = make(map[string]models.Metrics)
			pollCount = 0
		}
	}
}
