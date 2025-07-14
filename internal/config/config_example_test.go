package config

import (
	"fmt"
)

// ExampleConfig демонстрирует создание и использование структуры конфигурации.
func ExampleConfig() {
	// Создаем конфигурацию вручную
	cfg := &Config{
		RunAddr:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "my-secret-key",
	}

	fmt.Printf("Server address: %s\n", cfg.RunAddr)
	fmt.Printf("Store interval: %d seconds\n", cfg.StoreInterval)
	fmt.Printf("File storage path: %s\n", cfg.FileStoragePath)
	fmt.Printf("Restore on start: %t\n", cfg.Restore)
	fmt.Printf("Database configured: %t\n", cfg.DatabaseDSN != "")
	fmt.Printf("Key is set: %t\n", cfg.Key != "")

	// Output:
	// Server address: localhost:8080
	// Store interval: 300 seconds
	// File storage path: /tmp/metrics-db.json
	// Restore on start: true
	// Database configured: false
	// Key is set: true
}

// ExampleConfig_withDatabase демонстрирует конфигурацию с базой данных.
func ExampleConfig_withDatabase() {
	// Создаем конфигурацию с базой данных
	cfg := &Config{
		RunAddr:         "0.0.0.0:9090",
		StoreInterval:   600,
		FileStoragePath: "/var/metrics/data.json",
		Restore:         false,
		DatabaseDSN:     "postgres://user:pass@localhost:5432/metrics",
		Key:             "my-secret-key",
	}

	fmt.Printf("Server address: %s\n", cfg.RunAddr)
	fmt.Printf("Store interval: %d seconds\n", cfg.StoreInterval)
	fmt.Printf("File storage path: %s\n", cfg.FileStoragePath)
	fmt.Printf("Restore on start: %t\n", cfg.Restore)
	fmt.Printf("Database DSN: %s\n", cfg.DatabaseDSN)
	fmt.Printf("Database configured: %t\n", cfg.DatabaseDSN != "")
	fmt.Printf("Key is set: %t\n", cfg.Key != "")

	// Output:
	// Server address: 0.0.0.0:9090
	// Store interval: 600 seconds
	// File storage path: /var/metrics/data.json
	// Restore on start: false
	// Database DSN: postgres://user:pass@localhost:5432/metrics
	// Database configured: true
	// Key is set: true
}

// ExampleConfig_minimal демонстрирует минимальную конфигурацию.
func ExampleConfig_minimal() {
	// Создаем минимальную конфигурацию
	cfg := &Config{
		RunAddr:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "",
	}

	fmt.Printf("Server address: %s\n", cfg.RunAddr)
	fmt.Printf("Store interval: %d seconds\n", cfg.StoreInterval)
	fmt.Printf("File storage path: %s\n", cfg.FileStoragePath)
	fmt.Printf("Restore on start: %t\n", cfg.Restore)
	fmt.Printf("Database configured: %t\n", cfg.DatabaseDSN != "")
	fmt.Printf("Key is set: %t\n", cfg.Key != "")

	// Output:
	// Server address: localhost:8080
	// Store interval: 300 seconds
	// File storage path: /tmp/metrics-db.json
	// Restore on start: true
	// Database configured: false
	// Key is set: false
}
