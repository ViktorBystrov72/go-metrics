package server

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

// StorageManager управляет сохранением метрик
type StorageManager struct {
	storage storage.Storage
	config  *Config

	// Поля для graceful shutdown
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	stopped bool
}

// Config содержит конфигурацию для StorageManager
type Config struct {
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

// NewStorageManager создает новый StorageManager
func NewStorageManager(storage storage.Storage, config *Config) *StorageManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &StorageManager{
		storage: storage,
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start запускает периодическое сохранение
func (sm *StorageManager) Start() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.stopped {
		log.Printf("StorageManager уже остановлен")
		return
	}

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
		sm.wg.Add(1)
		go sm.periodicSave()
	}
}

// periodicSave выполняет периодическое сохранение с поддержкой graceful shutdown
func (sm *StorageManager) periodicSave() {
	defer sm.wg.Done()

	ticker := time.NewTicker(time.Duration(sm.config.StoreInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			log.Printf("Остановка периодического сохранения...")
			return
		case <-ticker.C:
			if err := sm.storage.SaveToFile(sm.config.FileStoragePath); err != nil {
				log.Printf("Ошибка при сохранении метрик: %v", err)
			}
		}
	}
}

// Stop gracefully останавливает StorageManager
func (sm *StorageManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.stopped {
		return
	}

	log.Printf("Остановка StorageManager...")
	sm.stopped = true

	// Сигнализируем всем горутинам о необходимости остановки
	sm.cancel()

	// Ожидаем завершения всех горутин
	sm.wg.Wait()

	log.Printf("StorageManager остановлен")
}

// Shutdown принудительно сохраняет данные и закрывает подключения
func (sm *StorageManager) Shutdown() error {
	log.Printf("Принудительное сохранение данных перед завершением...")

	// Если это не база данных, сохраняем в файл
	if !sm.storage.IsDatabase() && sm.config.FileStoragePath != "" {
		if err := sm.storage.SaveToFile(sm.config.FileStoragePath); err != nil {
			log.Printf("Ошибка при финальном сохранении: %v", err)
			return err
		}
		log.Printf("Данные успешно сохранены в: %s", sm.config.FileStoragePath)
	}

	// Закрываем подключение к базе данных, если оно есть
	if sm.storage.IsDatabase() {
		if closer, ok := sm.storage.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("Ошибка при закрытии базы данных: %v", err)
				return err
			}
			log.Printf("Подключение к базе данных закрыто")
		}
	}

	return nil
}
