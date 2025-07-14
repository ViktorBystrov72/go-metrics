package storage

import (
	"fmt"
	"os"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
)

// ExampleMemStorage_UpdateGauge демонстрирует обновление gauge метрики.
func ExampleMemStorage_UpdateGauge() {
	storage := NewMemStorage()

	// Обновляем gauge метрику
	storage.UpdateGauge("temperature", 23.5)
	storage.UpdateGauge("memory_usage", 85.2)

	// Получаем значение
	value, err := storage.GetGauge("temperature")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Temperature: %.1f\n", value)

	// Output:
	// Temperature: 23.5
}

// ExampleMemStorage_UpdateCounter демонстрирует обновление counter метрики.
func ExampleMemStorage_UpdateCounter() {
	storage := NewMemStorage()

	// Обновляем counter метрику
	storage.UpdateCounter("requests_total", 100)
	storage.UpdateCounter("requests_total", 50) // добавляется к предыдущему значению

	// Получаем значение
	value, err := storage.GetCounter("requests_total")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Total requests: %d\n", value)

	// Output:
	// Total requests: 150
}

// ExampleMemStorage_GetAllGauges демонстрирует получение всех gauge метрик.
func ExampleMemStorage_GetAllGauges() {
	storage := NewMemStorage()

	// Добавляем несколько gauge метрик
	storage.UpdateGauge("cpu_usage", 45.2)
	storage.UpdateGauge("memory_usage", 78.9)
	storage.UpdateGauge("disk_usage", 23.1)

	// Получаем все gauge метрики
	gauges := storage.GetAllGauges()

	fmt.Printf("Number of gauge metrics: %d\n", len(gauges))
	// Выводим в отсортированном порядке для стабильности теста
	fmt.Printf("cpu_usage: %.1f\n", gauges["cpu_usage"])
	fmt.Printf("memory_usage: %.1f\n", gauges["memory_usage"])
	fmt.Printf("disk_usage: %.1f\n", gauges["disk_usage"])

	// Output:
	// Number of gauge metrics: 3
	// cpu_usage: 45.2
	// memory_usage: 78.9
	// disk_usage: 23.1
}

// ExampleMemStorage_GetAllCounters демонстрирует получение всех counter метрик.
func ExampleMemStorage_GetAllCounters() {
	storage := NewMemStorage()

	// Добавляем несколько counter метрик
	storage.UpdateCounter("requests_total", 100)
	storage.UpdateCounter("errors_total", 5)
	storage.UpdateCounter("users_active", 25)

	// Получаем все counter метрики
	counters := storage.GetAllCounters()

	fmt.Printf("Number of counter metrics: %d\n", len(counters))
	// Выводим в отсортированном порядке для стабильности теста
	fmt.Printf("errors_total: %d\n", counters["errors_total"])
	fmt.Printf("requests_total: %d\n", counters["requests_total"])
	fmt.Printf("users_active: %d\n", counters["users_active"])

	// Output:
	// Number of counter metrics: 3
	// errors_total: 5
	// requests_total: 100
	// users_active: 25
}

// ExampleMemStorage_SaveToFile демонстрирует сохранение метрик в файл.
func ExampleMemStorage_SaveToFile() {
	storage := NewMemStorage()

	// Добавляем тестовые метрики
	storage.UpdateGauge("temperature", 23.5)
	storage.UpdateCounter("requests", 100)

	// Сохраняем в файл
	filename := "test_metrics.json"
	err := storage.SaveToFile(filename)
	if err != nil {
		fmt.Printf("Error saving: %v\n", err)
		return
	}

	// Проверяем, что файл создан
	if _, err := os.Stat(filename); err == nil {
		fmt.Printf("File saved successfully: %s\n", filename)
		// Удаляем тестовый файл
		os.Remove(filename)
	}

	// Output:
	// File saved successfully: test_metrics.json
}

// ExampleMemStorage_LoadFromFile демонстрирует загрузку метрик из файла.
func ExampleMemStorage_LoadFromFile() {
	storage := NewMemStorage()

	// Создаем тестовый файл с метриками
	filename := "test_load.json"
	testStorage := NewMemStorage()
	testStorage.UpdateGauge("loaded_temp", 25.0)
	testStorage.UpdateCounter("loaded_requests", 200)
	testStorage.SaveToFile(filename)

	// Загружаем метрики
	err := storage.LoadFromFile(filename)
	if err != nil {
		fmt.Printf("Error loading: %v\n", err)
		return
	}

	// Проверяем загруженные метрики
	gauges := storage.GetAllGauges()
	counters := storage.GetAllCounters()

	fmt.Printf("Loaded gauges: %d\n", len(gauges))
	fmt.Printf("Loaded counters: %d\n", len(counters))

	// Удаляем тестовый файл
	os.Remove(filename)

	// Output:
	// Loaded gauges: 1
	// Loaded counters: 1
}

// ExampleMemStorage_UpdateBatch демонстрирует массовое обновление метрик.
func ExampleMemStorage_UpdateBatch() {
	storage := NewMemStorage()

	// Создаем массив метрик для обновления
	metrics := []models.Metrics{
		{
			ID:    "batch_gauge1",
			MType: "gauge",
			Value: func() *float64 { v := 123.45; return &v }(),
		},
		{
			ID:    "batch_gauge2",
			MType: "gauge",
			Value: func() *float64 { v := 67.89; return &v }(),
		},
		{
			ID:    "batch_counter1",
			MType: "counter",
			Delta: func() *int64 { v := int64(10); return &v }(),
		},
		{
			ID:    "batch_counter2",
			MType: "counter",
			Delta: func() *int64 { v := int64(20); return &v }(),
		},
	}

	// Обновляем все метрики одной операцией
	err := storage.UpdateBatch(metrics)
	if err != nil {
		fmt.Printf("Error updating batch: %v\n", err)
		return
	}

	// Проверяем результаты
	gauges := storage.GetAllGauges()
	counters := storage.GetAllCounters()

	fmt.Printf("Batch update completed\n")
	fmt.Printf("Gauges updated: %d\n", len(gauges))
	fmt.Printf("Counters updated: %d\n", len(counters))

	// Output:
	// Batch update completed
	// Gauges updated: 2
	// Counters updated: 2
}

// ExampleMemStorage_Ping демонстрирует проверку доступности хранилища.
func ExampleMemStorage_Ping() {
	storage := NewMemStorage()

	// Проверяем доступность
	err := storage.Ping()
	if err != nil {
		fmt.Printf("Storage unavailable: %v\n", err)
		return
	}

	fmt.Printf("Storage is available\n")

	// Output:
	// Storage is available
}

// ExampleMemStorage_IsDatabase демонстрирует проверку типа хранилища.
func ExampleMemStorage_IsDatabase() {
	storage := NewMemStorage()

	isDB := storage.IsDatabase()
	fmt.Printf("Is database storage: %t\n", isDB)

	// Output:
	// Is database storage: false
}

// ExampleMemStorage_IsAvailable демонстрирует проверку доступности хранилища.
func ExampleMemStorage_IsAvailable() {
	storage := NewMemStorage()

	isAvailable := storage.IsAvailable()
	fmt.Printf("Storage is available: %t\n", isAvailable)

	// Output:
	// Storage is available: true
}
