package storage

import (
	"os"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
)

func TestMemStorage_SaveToFile_LoadFromFile(t *testing.T) {
	storage := NewMemStorage()

	storage.UpdateGauge("test_gauge", 123.45)
	storage.UpdateCounter("test_counter", 42)
	storage.UpdateCounter("test_counter", 8) // должно стать 50

	tempFile := "test_metrics.json"
	defer os.Remove(tempFile)

	// Тестируем сохранение
	err := storage.SaveToFile(tempFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Проверяем что файл создался
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Создаем новое хранилище и загружаем данные
	newStorage := NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Проверяем что данные загрузились корректно
	gauge, err := newStorage.GetGauge("test_gauge")
	if err != nil {
		t.Fatalf("Failed to get gauge after loading: %v", err)
	}
	if gauge != 123.45 {
		t.Errorf("Expected gauge value 123.45, got %f", gauge)
	}

	counter, err := newStorage.GetCounter("test_counter")
	if err != nil {
		t.Fatalf("Failed to get counter after loading: %v", err)
	}
	if counter != 50 {
		t.Errorf("Expected counter value 50, got %d", counter)
	}
}

func TestMemStorage_LoadFromFile_NonExistent(t *testing.T) {
	storage := NewMemStorage()

	// Пытаемся загрузить из несуществующего файла
	err := storage.LoadFromFile("non_existent_file.json")
	if err == nil {
		t.Error("Expected error when loading from non-existent file")
	}
}

func TestMemStorage_SaveToFile_Empty(t *testing.T) {
	storage := NewMemStorage()

	tempFile := "empty_metrics.json"
	defer os.Remove(tempFile)

	// Сохраняем пустое хранилище
	err := storage.SaveToFile(tempFile)
	if err != nil {
		t.Fatalf("SaveToFile failed for empty storage: %v", err)
	}

	// Загружаем в новое хранилище
	newStorage := NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Проверяем что хранилище осталось пустым
	gauges := newStorage.GetAllGauges()
	counters := newStorage.GetAllCounters()

	if len(gauges) != 0 {
		t.Errorf("Expected empty gauges, got %d items", len(gauges))
	}
	if len(counters) != 0 {
		t.Errorf("Expected empty counters, got %d items", len(counters))
	}
}

func TestMemStorage_SaveToFile_Concurrent(t *testing.T) {
	storage := NewMemStorage()

	// Добавляем метрики из нескольких горутин
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			storage.UpdateGauge("gauge_"+string(rune(id)), float64(id))
			storage.UpdateCounter("counter_"+string(rune(id)), int64(id))
			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	tempFile := "concurrent_metrics.json"
	defer os.Remove(tempFile)

	// Сохраняем и загружаем
	err := storage.SaveToFile(tempFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	newStorage := NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Проверяем что все метрики сохранились
	gauges := newStorage.GetAllGauges()
	counters := newStorage.GetAllCounters()

	if len(gauges) != 10 {
		t.Errorf("Expected 10 gauges, got %d", len(gauges))
	}
	if len(counters) != 10 {
		t.Errorf("Expected 10 counters, got %d", len(counters))
	}
}

func TestMemStorage_Ping(t *testing.T) {
	s := NewMemStorage()
	if err := s.Ping(); err != nil {
		t.Errorf("Ping() должен возвращать nil для MemStorage, получено: %v", err)
	}
}

func TestMemStorage_IsDatabase(t *testing.T) {
	s := NewMemStorage()
	if s.IsDatabase() {
		t.Error("IsDatabase() должен возвращать false для MemStorage")
	}
}

func TestMemStorage_IsAvailable(t *testing.T) {
	s := NewMemStorage()
	if !s.IsAvailable() {
		t.Error("IsAvailable() должен возвращать true для MemStorage")
	}
}

func TestMemStorage_UpdateBatch(t *testing.T) {
	s := NewMemStorage()
	metrics := []models.Metrics{
		{ID: "g1", MType: "gauge", Value: floatPtr(1.23)},
		{ID: "c1", MType: "counter", Delta: intPtr(10)},
	}
	err := s.UpdateBatch(metrics)
	if err != nil {
		t.Errorf("UpdateBatch() вернул ошибку: %v", err)
	}
	if v, _ := s.GetGauge("g1"); v != 1.23 {
		t.Errorf("UpdateBatch() не сохранил gauge")
	}
	if v, _ := s.GetCounter("c1"); v != 10 {
		t.Errorf("UpdateBatch() не сохранил counter")
	}
}

func TestMemStorage_UpdateBatch_Error(t *testing.T) {
	s := NewMemStorage()
	metrics := []models.Metrics{
		{ID: "bad", MType: "unknown"},
	}
	err := s.UpdateBatch(metrics)
	if err == nil {
		t.Error("UpdateBatch() должен вернуть ошибку для неизвестного типа метрики")
	}
}

func floatPtr(f float64) *float64 { return &f }
func intPtr(i int64) *int64       { return &i }
