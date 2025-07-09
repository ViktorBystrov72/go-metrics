package app

import (
	"context"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
)

// Collector определяет интерфейс для сбора метрик
type Collector interface {
	Start(ctx context.Context)
	Stop()
	Metrics() <-chan []models.Metrics
}

// Sender определяет интерфейс для отправки метрик
type Sender interface {
	Start(ctx context.Context)
	Stop()
	Metrics() chan<- []models.Metrics
}
