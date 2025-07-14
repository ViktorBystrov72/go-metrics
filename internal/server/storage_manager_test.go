package server

import (
	"testing"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

// TestNewStorageManager тестирует создание менеджера хранилища.
func TestNewStorageManager(t *testing.T) {
	storage := storage.NewMemStorage()
	config := &Config{
		StoreInterval:   1,
		FileStoragePath: "/tmp/test.json",
		Restore:         true,
	}
	manager := NewStorageManager(storage, config)
	if manager == nil {
		t.Fatal("NewStorageManager не должен возвращать nil")
	}
}

// TestStorageManagerStart тестирует запуск менеджера хранилища.
func TestStorageManagerStart(t *testing.T) {
	storage := storage.NewMemStorage()
	config := &Config{
		StoreInterval:   1,
		FileStoragePath: "/tmp/test.json",
		Restore:         true,
	}
	manager := NewStorageManager(storage, config)

	manager.Start()

	time.Sleep(10 * time.Millisecond)
}

// TestStorageManagerStartWithDatabase тестирует запуск с базой данных.
func TestStorageManagerStartWithDatabase(t *testing.T) {
	// Создаем заглушку для базы данных
	dbStorage, _ := storage.NewDatabaseStorage("test.db")
	config := &Config{
		StoreInterval:   1,
		FileStoragePath: "/tmp/test.json",
		Restore:         true,
	}
	manager := NewStorageManager(dbStorage, config)

	manager.Start()

	time.Sleep(10 * time.Millisecond)
}

// TestStorageManagerStartWithZeroInterval тестирует запуск с нулевым интервалом.
func TestStorageManagerStartWithZeroInterval(t *testing.T) {
	storage := storage.NewMemStorage()
	config := &Config{
		StoreInterval:   0,
		FileStoragePath: "/tmp/test.json",
		Restore:         true,
	}
	manager := NewStorageManager(storage, config)

	manager.Start()

	time.Sleep(10 * time.Millisecond)
}

// TestStorageManagerStartWithRestoreDisabled тестирует запуск без восстановления.
func TestStorageManagerStartWithRestoreDisabled(t *testing.T) {
	storage := storage.NewMemStorage()
	config := &Config{
		StoreInterval:   1,
		FileStoragePath: "/tmp/test.json",
		Restore:         false,
	}
	manager := NewStorageManager(storage, config)

	manager.Start()

	time.Sleep(10 * time.Millisecond)
}
