package config

import (
	"fmt"
	"os"
)

// ExampleConfig_Load демонстрирует загрузку конфигурации с значениями по умолчанию.
func ExampleConfig_Load() {
	// Загружаем конфигурацию
	cfg, err := Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Server address: %s\n", cfg.RunAddr)
	fmt.Printf("Store interval: %d seconds\n", cfg.StoreInterval)
	fmt.Printf("File storage path: %s\n", cfg.FileStoragePath)
	fmt.Printf("Restore on start: %t\n", cfg.Restore)

	// Output:
	// Server address: localhost:8080
	// Store interval: 300 seconds
	// File storage path: /tmp/metrics-db.json
	// Restore on start: true
}

// ExampleConfig_Load_withEnvironment демонстрирует загрузку конфигурации с переменными окружения.
func ExampleConfig_Load_withEnvironment() {
	// Устанавливаем переменные окружения
	os.Setenv("ADDRESS", "0.0.0.0:9090")
	os.Setenv("STORE_INTERVAL", "600")
	os.Setenv("FILE_STORAGE_PATH", "/var/metrics/data.json")
	os.Setenv("RESTORE", "false")
	os.Setenv("KEY", "my-secret-key")

	// Загружаем конфигурацию
	cfg, err := Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Server address: %s\n", cfg.RunAddr)
	fmt.Printf("Store interval: %d seconds\n", cfg.StoreInterval)
	fmt.Printf("File storage path: %s\n", cfg.FileStoragePath)
	fmt.Printf("Restore on start: %t\n", cfg.Restore)
	fmt.Printf("Key is set: %t\n", cfg.Key != "")

	// Очищаем переменные окружения
	os.Unsetenv("ADDRESS")
	os.Unsetenv("STORE_INTERVAL")
	os.Unsetenv("FILE_STORAGE_PATH")
	os.Unsetenv("RESTORE")
	os.Unsetenv("KEY")

	// Output:
	// Server address: 0.0.0.0:9090
	// Store interval: 600 seconds
	// File storage path: /var/metrics/data.json
	// Restore on start: false
	// Key is set: true
}

// ExampleConfig_Load_withDatabase демонстрирует загрузку конфигурации с настройками базы данных.
func ExampleConfig_Load_withDatabase() {
	// Устанавливаем переменную окружения для базы данных
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/metrics")

	// Загружаем конфигурацию
	cfg, err := Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Database DSN: %s\n", cfg.DatabaseDSN)
	fmt.Printf("Database is configured: %t\n", cfg.DatabaseDSN != "")

	// Очищаем переменную окружения
	os.Unsetenv("DATABASE_DSN")

	// Output:
	// Database DSN: postgres://user:pass@localhost:5432/metrics
	// Database is configured: true
}

// ExampleConfig_Load_invalidInterval демонстрирует обработку некорректного интервала сохранения.
func ExampleConfig_Load_invalidInterval() {
	// Устанавливаем некорректный интервал
	os.Setenv("STORE_INTERVAL", "-100")

	// Загружаем конфигурацию
	_, err := Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Очищаем переменную окружения
	os.Unsetenv("STORE_INTERVAL")

	// Output:
	// Error: STORE_INTERVAL must be non-negative, got -100
}
