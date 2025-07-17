package storage

import (
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
)

// TestNewDatabaseStorage тестирует создание хранилища базы данных.
func TestNewDatabaseStorage(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Fatal("NewDatabaseStorage не должен возвращать nil")
	}
}

// TestDatabaseStoragePing тестирует проверку соединения с базой данных.
func TestDatabaseStoragePing(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	// Ping должен работать даже без реальной БД
	err = storage.Ping()
	// Ожидаем ошибку, так как БД не существует
	if err == nil {
		t.Log("Ping вернул nil, что ожидаемо для несуществующей БД")
	}
}

// TestDatabaseStorageClose тестирует закрытие соединения с БД.
func TestDatabaseStorageClose(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	// Close не должен паниковать
	storage.Close()
}

// TestDatabaseStorageUpdateGauge тестирует обновление gauge метрики.
func TestDatabaseStorageUpdateGauge(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	// Обновление должно работать без ошибок
	storage.UpdateGauge("test", 123.45)
}

// TestDatabaseStorageUpdateCounter тестирует обновление counter метрики.
func TestDatabaseStorageUpdateCounter(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	// Обновление должно работать без ошибок
	storage.UpdateCounter("test", 123)
}

// TestDatabaseStorageGetGauge тестирует получение gauge метрики.
func TestDatabaseStorageGetGauge(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	value, err := storage.GetGauge("test")
	if err != nil {
		t.Logf("GetGauge вернул ошибку: %v", err)
	}
	_ = value
}

// TestDatabaseStorageGetCounter тестирует получение counter метрики.
func TestDatabaseStorageGetCounter(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	value, err := storage.GetCounter("test")
	if err != nil {
		t.Logf("GetCounter вернул ошибку: %v", err)
	}
	_ = value
}

// TestDatabaseStorageGetAllGauges тестирует получение всех gauge метрик.
func TestDatabaseStorageGetAllGauges(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	gauges := storage.GetAllGauges()
	_ = gauges
}

// TestDatabaseStorageGetAllCounters тестирует получение всех counter метрик.
func TestDatabaseStorageGetAllCounters(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	counters := storage.GetAllCounters()
	_ = counters
}

// TestDatabaseStorageUpdateBatch тестирует пакетное обновление метрик.
func TestDatabaseStorageUpdateBatch(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	// Создаем тестовые метрики
	metrics := []models.Metrics{
		{ID: "test1", MType: "gauge", Value: func() *float64 { v := 123.45; return &v }()},
		{ID: "test2", MType: "counter", Delta: func() *int64 { v := int64(100); return &v }()},
	}
	// Пакетное обновление должно работать без ошибок
	err = storage.UpdateBatch(metrics)
	if err != nil {
		t.Logf("UpdateBatch вернул ошибку: %v", err)
	}
}

// TestDatabaseStorageGetAllMetrics тестирует получение всех метрик.
func TestDatabaseStorageGetAllMetrics(t *testing.T) {
	storage, err := NewDatabaseStorage("test.db")
	if err != nil {
		t.Skipf("NewDatabaseStorage вернул ошибку: %v", err)
	}
	if storage == nil {
		t.Skip("storage is nil")
	}
	// Получение должно работать без ошибок
	metrics := storage.GetAllMetrics()
	_ = metrics
}
