package storage

import (
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
)

// BenchmarkDatabaseStorage_UpdateGauge тестирует производительность обновления gauge метрик
func BenchmarkDatabaseStorage_UpdateGauge(b *testing.B) {
	// Пропускаем тест если нет подключения к БД
	dsn := "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	storage, err := NewDatabaseStorage(dsn)
	if err != nil {
		b.Skipf("Database not available: %v", err)
	}
	defer storage.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			storage.UpdateGauge("benchmark_gauge", float64(i))
			i++
		}
	})
}

// BenchmarkDatabaseStorage_UpdateCounter тестирует производительность обновления counter метрик
func BenchmarkDatabaseStorage_UpdateCounter(b *testing.B) {
	dsn := "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	storage, err := NewDatabaseStorage(dsn)
	if err != nil {
		b.Skipf("Database not available: %v", err)
	}
	defer storage.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			storage.UpdateCounter("benchmark_counter", int64(i))
			i++
		}
	})
}

// BenchmarkDatabaseStorage_GetGauge тестирует производительность получения gauge метрик
func BenchmarkDatabaseStorage_GetGauge(b *testing.B) {
	dsn := "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	storage, err := NewDatabaseStorage(dsn)
	if err != nil {
		b.Skipf("Database not available: %v", err)
	}
	defer storage.Close()

	// Предварительно добавляем данные
	storage.UpdateGauge("benchmark_gauge_get", 123.45)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = storage.GetGauge("benchmark_gauge_get")
		}
	})
}

// BenchmarkDatabaseStorage_GetCounter тестирует производительность получения counter метрик
func BenchmarkDatabaseStorage_GetCounter(b *testing.B) {
	dsn := "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	storage, err := NewDatabaseStorage(dsn)
	if err != nil {
		b.Skipf("Database not available: %v", err)
	}
	defer storage.Close()

	// Предварительно добавляем данные
	storage.UpdateCounter("benchmark_counter_get", 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = storage.GetCounter("benchmark_counter_get")
		}
	})
}

// BenchmarkDatabaseStorage_UpdateBatch тестирует производительность batch обновлений
func BenchmarkDatabaseStorage_UpdateBatch(b *testing.B) {
	dsn := "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	storage, err := NewDatabaseStorage(dsn)
	if err != nil {
		b.Skipf("Database not available: %v", err)
	}
	defer storage.Close()

	// Подготавливаем batch данных
	metrics := make([]models.Metrics, 100)
	for i := 0; i < 100; i++ {
		value := float64(i)
		delta := int64(i)
		metrics[i] = models.Metrics{
			ID:    "batch_metric",
			MType: "gauge",
			Value: &value,
			Delta: &delta,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.UpdateBatch(metrics)
	}
}

// BenchmarkDatabaseStorage_GetAllGauges тестирует производительность получения всех gauge метрик
func BenchmarkDatabaseStorage_GetAllGauges(b *testing.B) {
	dsn := "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	storage, err := NewDatabaseStorage(dsn)
	if err != nil {
		b.Skipf("Database not available: %v", err)
	}
	defer storage.Close()

	// Предварительно добавляем данные
	for i := 0; i < 100; i++ {
		storage.UpdateGauge("benchmark_gauge_all", float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.GetAllGauges()
	}
}

// BenchmarkDatabaseStorage_GetAllCounters тестирует производительность получения всех counter метрик
func BenchmarkDatabaseStorage_GetAllCounters(b *testing.B) {
	dsn := "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	storage, err := NewDatabaseStorage(dsn)
	if err != nil {
		b.Skipf("Database not available: %v", err)
	}
	defer storage.Close()

	// Предварительно добавляем данные
	for i := 0; i < 100; i++ {
		storage.UpdateCounter("benchmark_counter_all", int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.GetAllCounters()
	}
}
