package config

import (
	"os"
	"testing"
)

func TestParseDurationToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{"1 second", "1s", 1, false},
		{"10 seconds", "10s", 10, false},
		{"1 minute", "1m", 60, false},
		{"1 hour", "1h", 3600, false},
		{"mixed", "1h30m", 5400, false},
		{"empty string", "", 0, true},
		{"invalid format", "invalid", 0, true},
		{"zero duration", "0s", 0, true},
		{"negative duration", "-5s", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDurationToSeconds(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Ожидалась ошибка для input '%s', но ошибки не было", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Неожиданная ошибка для input '%s': %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("Для input '%s' ожидался результат %d, получен %d", tt.input, tt.expected, result)
				}
			}
		})
	}
}

func TestLoadJSONFile(t *testing.T) {
	// Тест с валидным JSON файлом
	validJSON := `{
		"address": "localhost:9090",
		"report_interval": "5s",
		"poll_interval": "3s",
		"crypto_key": "/path/to/key.pem"
	}`

	// Создаем временный файл
	tmpfile, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем JSON в файл
	if _, err := tmpfile.Write([]byte(validJSON)); err != nil {
		t.Fatalf("Не удалось записать в временный файл: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Не удалось закрыть временный файл: %v", err)
	}

	// Тестируем загрузку конфигурации агента
	var agentConfig AgentJSONConfig
	err = LoadJSONFile(tmpfile.Name(), &agentConfig)
	if err != nil {
		t.Fatalf("Ошибка загрузки JSON файла: %v", err)
	}

	// Проверяем загруженные значения
	if agentConfig.Address == nil || *agentConfig.Address != "localhost:9090" {
		t.Errorf("Неверный address: ожидался 'localhost:9090', получен %v", agentConfig.Address)
	}
	if agentConfig.ReportInterval == nil || *agentConfig.ReportInterval != "5s" {
		t.Errorf("Неверный report_interval: ожидался '5s', получен %v", agentConfig.ReportInterval)
	}
	if agentConfig.PollInterval == nil || *agentConfig.PollInterval != "3s" {
		t.Errorf("Неверный poll_interval: ожидался '3s', получен %v", agentConfig.PollInterval)
	}
	if agentConfig.CryptoKey == nil || *agentConfig.CryptoKey != "/path/to/key.pem" {
		t.Errorf("Неверный crypto_key: ожидался '/path/to/key.pem', получен %v", agentConfig.CryptoKey)
	}

	// Тест с несуществующим файлом
	err = LoadJSONFile("nonexistent.json", &agentConfig)
	if err == nil {
		t.Error("Ожидалась ошибка для несуществующего файла")
	}

	err = LoadJSONFile("", &agentConfig)
	if err != nil {
		t.Errorf("Неожиданная ошибка для пустого имени файла: %v", err)
	}
}

func TestServerJSONConfigApply(t *testing.T) {
	// Тестируем применение JSON конфигурации сервера
	jsonConfig := &ServerJSONConfig{
		Address:       stringPtr("json-address:8080"),
		Restore:       boolPtr(false),
		StoreInterval: stringPtr("10s"),
		StoreFile:     stringPtr("/json/path"),
		DatabaseDSN:   stringPtr("json-dsn"),
		CryptoKey:     stringPtr("/json/key.pem"),
	}

	finalConfig := &ServerJSONConfig{
		Address: stringPtr("flag-address:9090"), // Этот не должен быть перезаписан
		// Остальные значения должны быть взяты из jsonConfig
	}

	jsonConfig.ApplyToServerConfig(finalConfig)

	// Address не должен измениться (приоритет у finalConfig)
	if finalConfig.Address == nil || *finalConfig.Address != "flag-address:9090" {
		t.Errorf("Address не должен был измениться, получен: %v", finalConfig.Address)
	}

	// Остальные значения должны быть взяты из jsonConfig
	if finalConfig.Restore == nil || *finalConfig.Restore != false {
		t.Errorf("Restore должен быть false, получен: %v", finalConfig.Restore)
	}
	if finalConfig.StoreInterval == nil || *finalConfig.StoreInterval != "10s" {
		t.Errorf("StoreInterval должен быть '10s', получен: %v", finalConfig.StoreInterval)
	}
	if finalConfig.StoreFile == nil || *finalConfig.StoreFile != "/json/path" {
		t.Errorf("StoreFile должен быть '/json/path', получен: %v", finalConfig.StoreFile)
	}
	if finalConfig.DatabaseDSN == nil || *finalConfig.DatabaseDSN != "json-dsn" {
		t.Errorf("DatabaseDSN должен быть 'json-dsn', получен: %v", finalConfig.DatabaseDSN)
	}
	if finalConfig.CryptoKey == nil || *finalConfig.CryptoKey != "/json/key.pem" {
		t.Errorf("CryptoKey должен быть '/json/key.pem', получен: %v", finalConfig.CryptoKey)
	}
}

func TestAgentJSONConfigApply(t *testing.T) {
	// Тестируем применение JSON конфигурации агента
	jsonConfig := &AgentJSONConfig{
		Address:        stringPtr("json-address:8080"),
		ReportInterval: stringPtr("15s"),
		PollInterval:   stringPtr("5s"),
		CryptoKey:      stringPtr("/json/key.pem"),
	}

	finalConfig := &AgentJSONConfig{
		Address: stringPtr("flag-address:9090"), // Этот не должен быть перезаписан
		// Остальные значения должны быть взяты из jsonConfig
	}

	jsonConfig.ApplyToAgentConfig(finalConfig)

	// Address не должен измениться (приоритет у finalConfig)
	if finalConfig.Address == nil || *finalConfig.Address != "flag-address:9090" {
		t.Errorf("Address не должен был измениться, получен: %v", finalConfig.Address)
	}

	// Остальные значения должны быть взяты из jsonConfig
	if finalConfig.ReportInterval == nil || *finalConfig.ReportInterval != "15s" {
		t.Errorf("ReportInterval должен быть '15s', получен: %v", finalConfig.ReportInterval)
	}
	if finalConfig.PollInterval == nil || *finalConfig.PollInterval != "5s" {
		t.Errorf("PollInterval должен быть '5s', получен: %v", finalConfig.PollInterval)
	}
	if finalConfig.CryptoKey == nil || *finalConfig.CryptoKey != "/json/key.pem" {
		t.Errorf("CryptoKey должен быть '/json/key.pem', получен: %v", finalConfig.CryptoKey)
	}
}

func TestInvalidJSON(t *testing.T) {
	// Тест с некорректным JSON
	invalidJSON := `{
		"address": "localhost:9090",
		"report_interval": "5s",
		"invalid_json":
	}`

	tmpfile, err := os.CreateTemp("", "test-invalid-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(invalidJSON)); err != nil {
		t.Fatalf("Не удалось записать в временный файл: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Не удалось закрыть временный файл: %v", err)
	}

	var agentConfig AgentJSONConfig
	err = LoadJSONFile(tmpfile.Name(), &agentConfig)
	if err == nil {
		t.Error("Ожидалась ошибка для некорректного JSON")
	}
}
