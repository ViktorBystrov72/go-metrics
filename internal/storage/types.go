package storage

// MetricType тип метрики
type MetricType string

const (
	Gauge   MetricType = "gauge"
	Counter MetricType = "counter"
)
