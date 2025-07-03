package storage

// Storage интерфейс для хранения метрик
type Storage interface {
	// UpdateGauge обновляет значение gauge метрики
	UpdateGauge(name string, value float64)

	// UpdateCounter обновляет значение counter метрики
	UpdateCounter(name string, value int64)

	// GetGauge возвращает значение gauge метрики
	GetGauge(name string) (float64, bool)

	// GetCounter возвращает значение counter метрики
	GetCounter(name string) (int64, bool)

	// GetAllGauges возвращает все gauge метрики
	GetAllGauges() map[string]float64

	// GetAllCounters возвращает все counter метрики
	GetAllCounters() map[string]int64

	// SaveToFile сохраняет метрики в файл
	SaveToFile(filename string) error

	// LoadFromFile загружает метрики из файла
	LoadFromFile(filename string) error
}
