package tests

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/app"
	"github.com/ViktorBystrov72/go-metrics/internal/crypto"
	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/server"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

func TestCryptoIntegration(t *testing.T) {
	// Создаем временные файлы для ключей
	privateKeyFile := "test_integration_private.pem"
	publicKeyFile := "test_integration_public.pem"

	defer func() {
		os.Remove(privateKeyFile)
		os.Remove(publicKeyFile)
	}()

	// Генерируем ключи
	privateKey, publicKey, err := crypto.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	// Сохраняем ключи в файлы
	err = crypto.SavePrivateKeyToFile(privateKeyFile, privateKey)
	if err != nil {
		t.Fatalf("Ошибка сохранения приватного ключа: %v", err)
	}

	err = crypto.SavePublicKeyToFile(publicKeyFile, publicKey)
	if err != nil {
		t.Fatalf("Ошибка сохранения публичного ключа: %v", err)
	}

	// Создаем сервер БЕЗ дешифрования для начала
	storage := storage.NewMemStorage()
	router := server.NewRouter(storage, "", "") // без ключей
	testServer := httptest.NewServer(router.GetRouter())
	defer testServer.Close()

	// Создаем конфигурацию агента БЕЗ шифрования для начала
	cfg := &app.AgentConfig{
		Address:        testServer.URL[7:], // убираем "http://"
		ReportInterval: 1,
		PollInterval:   1,
		Key:            "", // без хеширования
		RateLimit:      1,
		CryptoKey:      "", // без шифрования
	}

	// Создаем агент
	sender := app.NewMetricsSender(cfg)

	// Тестовые метрики (без хешей)
	testMetrics := []models.Metrics{
		{
			ID:    "test_gauge",
			MType: "gauge",
			Value: func() *float64 { v := 123.45; return &v }(),
		},
	}

	// Получаем интерфейс MetricsSender и приводим к конкретному типу
	metricsSender, ok := sender.(*app.MetricsSender)
	if !ok {
		t.Fatalf("Не удалось привести sender к типу *MetricsSender")
	}

	// Отправляем метрики БЕЗ шифрования
	err = metricsSender.SendMetricsBatch(testMetrics)
	if err != nil {
		t.Fatalf("Ошибка отправки метрик без шифрования: %v", err)
	}

	// Проверяем, что метрики были успешно получены
	value, err := storage.GetGauge("test_gauge")
	if err != nil {
		t.Errorf("Ошибка получения gauge метрики: %v", err)
	} else if value != 123.45 {
		t.Errorf("Неверное значение gauge метрики: ожидалось 123.45, получено %f", value)
	}

	t.Log("Тест без шифрования прошел успешно")
}

func TestCryptoIntegrationWithEncryption(t *testing.T) {
	privateKeyFile := "test_integration_private2.pem"
	publicKeyFile := "test_integration_public2.pem"

	defer func() {
		os.Remove(privateKeyFile)
		os.Remove(publicKeyFile)
	}()

	// Генерируем ключи
	privateKey, publicKey, err := crypto.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	// Сохраняем ключи в файлы
	err = crypto.SavePrivateKeyToFile(privateKeyFile, privateKey)
	if err != nil {
		t.Fatalf("Ошибка сохранения приватного ключа: %v", err)
	}

	err = crypto.SavePublicKeyToFile(publicKeyFile, publicKey)
	if err != nil {
		t.Fatalf("Ошибка сохранения публичного ключа: %v", err)
	}

	// Создаем сервер С дешифрованием
	storage := storage.NewMemStorage()
	router := server.NewRouter(storage, "", privateKeyFile) // с приватным ключом
	testServer := httptest.NewServer(router.GetRouter())
	defer testServer.Close()

	// Создаем конфигурацию агента С шифрованием
	cfg := &app.AgentConfig{
		Address:        testServer.URL[7:], // убираем "http://"
		ReportInterval: 1,
		PollInterval:   1,
		Key:            "", // без хеширования пока
		RateLimit:      1,
		CryptoKey:      publicKeyFile, // с шифрованием
	}

	// Создаем агент
	sender := app.NewMetricsSender(cfg)

	// Тестовые метрики (без хешей)
	testMetrics := []models.Metrics{
		{
			ID:    "test_gauge_encrypted",
			MType: "gauge",
			Value: func() *float64 { v := 456.78; return &v }(),
		},
	}

	metricsSender, ok := sender.(*app.MetricsSender)
	if !ok {
		t.Fatalf("Не удалось привести sender к типу *MetricsSender")
	}

	// Отправляем метрики С шифрованием
	err = metricsSender.SendMetricsBatch(testMetrics)
	if err != nil {
		t.Fatalf("Ошибка отправки зашифрованных метрик: %v", err)
	}

	// Проверяем, что метрики были успешно получены и дешифрованы
	value, err := storage.GetGauge("test_gauge_encrypted")
	if err != nil {
		t.Errorf("Ошибка получения зашифрованной gauge метрики: %v", err)
	} else if value != 456.78 {
		t.Errorf("Неверное значение зашифрованной gauge метрики: ожидалось 456.78, получено %f", value)
	}

	t.Log("Тест с шифрованием прошел успешно")
}
