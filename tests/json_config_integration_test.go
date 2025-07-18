package tests

import (
	"os"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/config"
)

func TestServerJSONConfigIntegration(t *testing.T) {
	// Создаем временный файл конфигурации сервера
	serverConfigJSON := `{
		"address": "localhost:9999",
		"restore": false,
		"store_interval": "10s",
		"store_file": "/tmp/test-metrics.json",
		"database_dsn": "test-dsn",
		"crypto_key": "/path/to/server/key.pem"
	}`

	serverFile, err := os.CreateTemp("", "server-config-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл для сервера: %v", err)
	}
	defer os.Remove(serverFile.Name())

	if _, err := serverFile.Write([]byte(serverConfigJSON)); err != nil {
		t.Fatalf("Ошибка записи в файл сервера: %v", err)
	}
	serverFile.Close()

	// Сохраняем и очищаем переменные окружения
	origConfig := os.Getenv("CONFIG")
	defer func() {
		if origConfig != "" {
			os.Setenv("CONFIG", origConfig)
		} else {
			os.Unsetenv("CONFIG")
		}
	}()

	// Устанавливаем файл конфигурации
	os.Setenv("CONFIG", serverFile.Name())

	serverConfig, err := config.Load()
	if err != nil {
		t.Fatalf("Ошибка загрузки конфигурации сервера: %v", err)
	}

	// Проверяем значения сервера
	if serverConfig.RunAddr != "localhost:9999" {
		t.Errorf("Неверный RunAddr: ожидался 'localhost:9999', получен '%s'", serverConfig.RunAddr)
	}
	if serverConfig.Restore != false {
		t.Errorf("Неверный Restore: ожидался false, получен %v", serverConfig.Restore)
	}
	if serverConfig.StoreInterval != 10 {
		t.Errorf("Неверный StoreInterval: ожидался 10, получен %d", serverConfig.StoreInterval)
	}
	if serverConfig.FileStoragePath != "/tmp/test-metrics.json" {
		t.Errorf("Неверный FileStoragePath: ожидался '/tmp/test-metrics.json', получен '%s'", serverConfig.FileStoragePath)
	}
	if serverConfig.DatabaseDSN != "test-dsn" {
		t.Errorf("Неверный DatabaseDSN: ожидался 'test-dsn', получен '%s'", serverConfig.DatabaseDSN)
	}
	if serverConfig.CryptoKey != "/path/to/server/key.pem" {
		t.Errorf("Неверный CryptoKey: ожидался '/path/to/server/key.pem', получен '%s'", serverConfig.CryptoKey)
	}

	t.Log("Тест JSON конфигурации сервера прошел успешно")
}

func TestServerJSONConfigPriority(t *testing.T) {
	// Тестируем приоритет: env > JSON для сервера
	jsonConfig := `{
		"address": "json:8080",
		"restore": true,
		"store_interval": "30s",
		"crypto_key": "/json/key.pem"
	}`

	configFile, err := os.CreateTemp("", "priority-test-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.Write([]byte(jsonConfig)); err != nil {
		t.Fatalf("Ошибка записи в файл: %v", err)
	}
	configFile.Close()

	// Сохраняем оригинальные env переменные
	origConfig := os.Getenv("CONFIG")
	origAddress := os.Getenv("ADDRESS")
	origRestore := os.Getenv("RESTORE")

	defer func() {
		// Восстанавливаем оригинальные значения
		if origConfig != "" {
			os.Setenv("CONFIG", origConfig)
		} else {
			os.Unsetenv("CONFIG")
		}
		if origAddress != "" {
			os.Setenv("ADDRESS", origAddress)
		} else {
			os.Unsetenv("ADDRESS")
		}
		if origRestore != "" {
			os.Setenv("RESTORE", origRestore)
		} else {
			os.Unsetenv("RESTORE")
		}
	}()

	// Устанавливаем переменные окружения (должны переопределить JSON)
	os.Setenv("CONFIG", configFile.Name())
	os.Setenv("ADDRESS", "env:8080")
	os.Setenv("RESTORE", "false")

	serverConfig, err := config.Load()
	if err != nil {
		t.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Проверяем приоритеты
	// Address: env должен переопределить JSON
	if serverConfig.RunAddr != "env:8080" {
		t.Errorf("Неверный приоритет для Address: ожидался 'env:8080', получен '%s'", serverConfig.RunAddr)
	}

	// Restore: env должен переопределить JSON
	if serverConfig.Restore != false {
		t.Errorf("Неверный приоритет для Restore: ожидался false, получен %v", serverConfig.Restore)
	}

	// StoreInterval и CryptoKey: JSON должен применяться (env не заданы)
	if serverConfig.StoreInterval != 30 {
		t.Errorf("Неверный приоритет для StoreInterval: ожидался 30, получен %d", serverConfig.StoreInterval)
	}
	if serverConfig.CryptoKey != "/json/key.pem" {
		t.Errorf("Неверный приоритет для CryptoKey: ожидался '/json/key.pem', получен '%s'", serverConfig.CryptoKey)
	}

	t.Log("Тест приоритетов конфигурации сервера прошел успешно")
}
