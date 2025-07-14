package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	os.Setenv("ADDRESS", ":9999")
	os.Setenv("STORE_INTERVAL", "5")
	os.Setenv("FILE_STORAGE_PATH", "test.json")
	os.Setenv("RESTORE", "true")
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/db")
	os.Setenv("KEY", "testkey")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.RunAddr != ":9999" || cfg.StoreInterval != 5 || cfg.FileStoragePath != "test.json" || !cfg.Restore || cfg.DatabaseDSN != "postgres://user:pass@localhost:5432/db" || cfg.Key != "testkey" {
		t.Errorf("Load() неверно парсит переменные окружения")
	}
}
