package app

import (
	"os"
	"testing"
)

func TestParseAgentConfig(t *testing.T) {
	os.Setenv("ADDRESS", "localhost:9999")
	os.Setenv("REPORT_INTERVAL", "5")
	os.Setenv("POLL_INTERVAL", "3")
	os.Setenv("RATE_LIMIT", "2")
	os.Setenv("KEY", "testkey")
	cfg, err := ParseAgentConfig()
	if err != nil {
		t.Fatalf("ParseAgentConfig() error: %v", err)
	}
	if cfg.Address != "localhost:9999" || cfg.ReportInterval != 5 || cfg.PollInterval != 3 || cfg.RateLimit != 2 || cfg.Key != "testkey" {
		t.Errorf("ParseAgentConfig() неверно парсит переменные окружения")
	}
}

func TestNewMetricsCollector(t *testing.T) {
	cfg := &AgentConfig{PollInterval: 1, Key: ""}
	collector := NewMetricsCollector(cfg)
	if collector == nil {
		t.Error("NewMetricsCollector() вернул nil")
	}
}

func TestNewMetricsSender(t *testing.T) {
	cfg := &AgentConfig{Address: "localhost:8080", RateLimit: 1, Key: ""}
	sender := NewMetricsSender(cfg)
	if sender == nil {
		t.Error("NewMetricsSender() вернул nil")
	}
}

func TestNewMetric(t *testing.T) {
	val := 1.23
	m := NewMetric("id", "gauge", &val, nil, "")
	if m.ID != "id" || m.MType != "gauge" || m.Value == nil {
		t.Error("NewMetric() создал некорректную метрику")
	}
}
