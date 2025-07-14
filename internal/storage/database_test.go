package storage

import (
	"testing"
)

func TestNewDatabaseStorage(t *testing.T) {
	// Тест с неверным DSN
	_, err := NewDatabaseStorage("invalid-dsn")
	if err == nil {
		t.Error("NewDatabaseStorage() должен вернуть ошибку для неверного DSN")
	}
}

func TestDatabaseStorage_IsDatabase(t *testing.T) {
	storage := &DatabaseStorage{}
	if !storage.IsDatabase() {
		t.Error("IsDatabase() должен вернуть true для DatabaseStorage")
	}
}

func TestDatabaseStorage_IsAvailable(t *testing.T) {
	storage := &DatabaseStorage{}
	if storage.IsAvailable() {
		t.Error("IsAvailable() должен вернуть false для DatabaseStorage без подключения")
	}
}

func TestDatabaseStorage_Ping(t *testing.T) {
	// Пропускаем тест Ping, так как он требует подключения к БД
	t.Skip("Ping тест пропущен - требует подключения к БД")
}

func TestDatabaseStorage_SaveToFile(t *testing.T) {
	storage := &DatabaseStorage{}
	err := storage.SaveToFile("/tmp/test.json")
	if err != nil {
		t.Errorf("SaveToFile() не должен возвращать ошибку для DatabaseStorage, получено: %v", err)
	}
}

func TestDatabaseStorage_LoadFromFile(t *testing.T) {
	storage := &DatabaseStorage{}
	err := storage.LoadFromFile("/tmp/test.json")
	if err != nil {
		t.Errorf("LoadFromFile() не должен возвращать ошибку для DatabaseStorage, получено: %v", err)
	}
}
