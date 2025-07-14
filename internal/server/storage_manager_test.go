package server

import (
	"testing"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

func TestNewStorageManager(t *testing.T) {
	storage := storage.NewMemStorage()
	config := &Config{
		StoreInterval:   1,
		FileStoragePath: "/tmp/test.json",
		Restore:         true,
	}
	manager := NewStorageManager(storage, config)
	if manager == nil {
		t.Error("NewStorageManager() вернул nil")
	}
}

func TestStorageManager_Start(t *testing.T) {
	storage := storage.NewMemStorage()
	config := &Config{
		StoreInterval:   1,
		FileStoragePath: "/tmp/test.json",
		Restore:         true,
	}
	manager := NewStorageManager(storage, config)

	manager.Start()

	time.Sleep(50 * time.Millisecond)
}
