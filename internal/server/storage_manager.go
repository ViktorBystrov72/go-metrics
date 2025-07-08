package server

import (
	"log"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

// StorageManager управляет сохранением метрик
type StorageManager struct {
	storage storage.Storage
	config  *Config
}

// Config содержит конфигурацию для StorageManager
type Config struct {
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

// NewStorageManager создает новый StorageManager
func NewStorageManager(storage storage.Storage, config *Config) *StorageManager {
	return &StorageManager{
		storage: storage,
		config:  config,
	}
}

// Start запускает периодическое сохранение
func (sm *StorageManager) Start() {
	if sm.storage.IsDatabase() {
		log.Printf("Database storage detected, skipping file operations")
		return
	}

	// Для файлового хранилища загружаем данные при запуске
	if sm.config.Restore {
		if err := sm.storage.LoadFromFile(sm.config.FileStoragePath); err != nil {
			log.Printf("Failed to load from file: %v", err)
		} else {
			log.Printf("Loaded metrics from file: %s", sm.config.FileStoragePath)
		}
	}

	// Периодическое сохранение только для файлового хранилища
	if sm.config.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(sm.config.StoreInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := sm.storage.SaveToFile(sm.config.FileStoragePath); err != nil {
					log.Printf("Ошибка при сохранении метрик: %v", err)
				}
			}
		}()
	}
}
