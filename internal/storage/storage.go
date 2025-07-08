package storage

import "github.com/ViktorBystrov72/go-metrics/internal/models"

// Storage интерфейс для хранения метрик
type Storage interface {
	// UpdateGauge обновляет значение gauge метрики
	UpdateGauge(name string, value float64)

	// UpdateCounter обновляет значение counter метрики
	UpdateCounter(name string, value int64)

	// GetGauge возвращает значение gauge метрики
	GetGauge(name string) (float64, error)

	// GetCounter возвращает значение counter метрики
	GetCounter(name string) (int64, error)

	// GetAllGauges возвращает все gauge метрики
	GetAllGauges() map[string]float64

	// GetAllCounters возвращает все counter метрики
	GetAllCounters() map[string]int64

	// SaveToFile сохраняет метрики в файл
	SaveToFile(filename string) error

	// LoadFromFile загружает метрики из файла
	LoadFromFile(filename string) error

	// Ping проверяет соединение с хранилищем (для БД)
	Ping() error

	// IsDatabase возвращает true, если это база данных
	IsDatabase() bool

	// IsAvailable возвращает true, если хранилище доступно
	IsAvailable() bool

	// UpdateBatch обновляет множество метрик в одной операции
	UpdateBatch(metrics []models.Metrics) error
}
