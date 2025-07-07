package storage

import (
	"os"
	"testing"
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
	if err != nil || gauge != 123.45 {
		t.Errorf("Expected gauge value 123.45, got %f, error: %v", gauge, err)
	}

	counter, err := newStorage.GetCounter("test_counter")
	if err != nil || counter != 50 {
		t.Errorf("Expected counter value 50, got %d, error: %v", counter, err)
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
