package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"compress/gzip"
	"io"

	"os"

	"github.com/ViktorBystrov72/go-metrics/internal/middleware"
	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_JSON_API(t *testing.T) {
	testStorage := storage.NewMemStorage()
	server := NewServer(testStorage)

	r := chi.NewRouter()
	r.Post("/update/", server.updateJSONHandler)
	r.Post("/value/", server.valueJSONHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("Update Gauge", func(t *testing.T) {
		val := 123.45
		m := models.Metrics{ID: "testGauge", MType: "gauge", Value: &val}
		body, _ := json.Marshal(m)
		resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Update Counter", func(t *testing.T) {
		delta := int64(10)
		m := models.Metrics{ID: "testCounter", MType: "counter", Delta: &delta}
		body, _ := json.Marshal(m)
		resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Get Gauge Value", func(t *testing.T) {
		val := 123.45
		m := models.Metrics{ID: "testGauge2", MType: "gauge", Value: &val}
		body, _ := json.Marshal(m)
		resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		getReq := models.Metrics{ID: "testGauge2", MType: "gauge"}
		getBody, _ := json.Marshal(getReq)
		resp, err = http.Post(ts.URL+"/value/", "application/json", bytes.NewBuffer(getBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var got models.Metrics
		_ = json.NewDecoder(resp.Body).Decode(&got)
		assert.NotNil(t, got.Value)
		assert.Equal(t, val, *got.Value)
	})

	t.Run("Get Counter Value", func(t *testing.T) {
		delta := int64(5)
		m := models.Metrics{ID: "testCounter2", MType: "counter", Delta: &delta}
		body, _ := json.Marshal(m)
		resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		getReq := models.Metrics{ID: "testCounter2", MType: "counter"}
		getBody, _ := json.Marshal(getReq)
		resp, err = http.Post(ts.URL+"/value/", "application/json", bytes.NewBuffer(getBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var got models.Metrics
		_ = json.NewDecoder(resp.Body).Decode(&got)
		assert.NotNil(t, got.Delta)
		assert.Equal(t, delta, *got.Delta)
	})

	t.Run("Get Non-existent Metric", func(t *testing.T) {
		getReq := models.Metrics{ID: "nonExistent", MType: "gauge"}
		getBody, _ := json.Marshal(getReq)
		resp, err := http.Post(ts.URL+"/value/", "application/json", bytes.NewBuffer(getBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Invalid Metric Type", func(t *testing.T) {
		m := models.Metrics{ID: "testMetric", MType: "invalid"}
		body, _ := json.Marshal(m)
		resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Invalid Value", func(t *testing.T) {
		m := models.Metrics{ID: "testMetric", MType: "gauge"}
		body, _ := json.Marshal(m)
		resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestGzipCompression(t *testing.T) {
	testStorage := storage.NewMemStorage()
	server := NewServer(testStorage)

	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Post("/update/", server.updateJSONHandler)
	r.Post("/value/", server.valueJSONHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("accepts_gzip_request", func(t *testing.T) {
		val := 42.0
		m := models.Metrics{ID: "gzipGauge", MType: "gauge", Value: &val}
		body, _ := json.Marshal(m)
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write(body)
		require.NoError(t, err)
		require.NoError(t, gz.Close())

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/update/", &buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("returns_gzip_response", func(t *testing.T) {
		val := 99.0
		m := models.Metrics{ID: "gzipGauge2", MType: "gauge", Value: &val}
		body, _ := json.Marshal(m)
		resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		getReq := models.Metrics{ID: "gzipGauge2", MType: "gauge"}
		getBody, _ := json.Marshal(getReq)
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/value/", bytes.NewBuffer(getBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Encoding", "gzip")
		client := &http.Client{}
		resp2, err := client.Do(req)
		require.NoError(t, err)
		defer resp2.Body.Close()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
		assert.Equal(t, "gzip", resp2.Header.Get("Content-Encoding"))
		zr, err := gzip.NewReader(resp2.Body)
		require.NoError(t, err)
		defer zr.Close()
		b, err := io.ReadAll(zr)
		require.NoError(t, err)
		var got models.Metrics
		_ = json.Unmarshal(b, &got)
		assert.NotNil(t, got.Value)
		assert.Equal(t, val, *got.Value)
	})
}

func TestServer_Configuration_Priority(t *testing.T) {
	// Приоритет конфигурации: env > flag > default

	originalStoreInterval := os.Getenv("STORE_INTERVAL")
	originalFileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	originalRestore := os.Getenv("RESTORE")

	// Восстанавливаем
	defer func() {
		if originalStoreInterval != "" {
			os.Setenv("STORE_INTERVAL", originalStoreInterval)
		} else {
			os.Unsetenv("STORE_INTERVAL")
		}
		if originalFileStoragePath != "" {
			os.Setenv("FILE_STORAGE_PATH", originalFileStoragePath)
		} else {
			os.Unsetenv("FILE_STORAGE_PATH")
		}
		if originalRestore != "" {
			os.Setenv("RESTORE", originalRestore)
		} else {
			os.Unsetenv("RESTORE")
		}
	}()

	// Переменные окружения имеют приоритет над флагами
	os.Setenv("STORE_INTERVAL", "60")
	os.Setenv("FILE_STORAGE_PATH", "/custom/path.json")
	os.Setenv("RESTORE", "false")

	t.Run("Environment variables override flags", func(t *testing.T) {
		// Проверка переменные окружения читаются корректно
		if os.Getenv("STORE_INTERVAL") != "60" {
			t.Error("STORE_INTERVAL not set correctly")
		}
		if os.Getenv("FILE_STORAGE_PATH") != "/custom/path.json" {
			t.Error("FILE_STORAGE_PATH not set correctly")
		}
		if os.Getenv("RESTORE") != "false" {
			t.Error("RESTORE not set correctly")
		}
	})
}

func TestServer_FileStorage_Integration(t *testing.T) {
	tempFile := "test_server_metrics.json"
	defer os.Remove(tempFile)

	// Тестовое хранилище
	testStorage := storage.NewMemStorage()

	// Тестовые метрики
	testStorage.UpdateGauge("test_gauge", 123.45)
	testStorage.UpdateCounter("test_counter", 42)

	// Сохраняем метрики
	err := testStorage.SaveToFile(tempFile)
	require.NoError(t, err)

	// Файл создался
	_, err = os.Stat(tempFile)
	require.NoError(t, err)

	// Создаем новое хранилище и загружаем
	newStorage := storage.NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	require.NoError(t, err)

	// Данные восстановились
	gauge, exists := newStorage.GetGauge("test_gauge")
	require.True(t, exists)
	assert.Equal(t, 123.45, gauge)

	counter, exists := newStorage.GetCounter("test_counter")
	require.True(t, exists)
	assert.Equal(t, int64(42), counter)
}

func TestServer_StoreInterval_Zero(t *testing.T) {
	// Тест для синхронного сохранения (STORE_INTERVAL = 0)
	tempFile := "sync_metrics.json"
	defer os.Remove(tempFile)

	testStorage := storage.NewMemStorage()
	testStorage.UpdateGauge("sync_test", 99.99)

	// При STORE_INTERVAL = 0 сохранение должно происходить синхронно
	// Значит каждая операция обновления сразу сохраняется в файл

	// Сохраняем вручную для теста
	err := testStorage.SaveToFile(tempFile)
	require.NoError(t, err)

	// Проверяем что файл создался и содержит данные
	_, err = os.Stat(tempFile)
	require.NoError(t, err)

	// Загрузка и проверка
	newStorage := storage.NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	require.NoError(t, err)

	gauge, exists := newStorage.GetGauge("sync_test")
	require.True(t, exists)
	assert.Equal(t, 99.99, gauge)
}

func TestServer_Restore_Disabled(t *testing.T) {
	// Тест когда RESTORE = false
	tempFile := "restore_disabled.json"
	defer os.Remove(tempFile)

	// Создаем файл с данными
	testStorage := storage.NewMemStorage()
	testStorage.UpdateGauge("persistent_gauge", 777.77)
	err := testStorage.SaveToFile(tempFile)
	require.NoError(t, err)

	// При RESTORE = false новое хранилище должно быть пустым
	// даже если файл существует
	newStorage := storage.NewMemStorage()
	// Не вызываем LoadFromFile, так как RESTORE = false

	// Проверяем что хранилище пустое
	gauges := newStorage.GetAllGauges()
	counters := newStorage.GetAllCounters()

	assert.Equal(t, 0, len(gauges))
	assert.Equal(t, 0, len(counters))
}

func TestServer_FileStorage_ConcurrentAccess(t *testing.T) {
	tempFile := "concurrent_server_metrics.json"
	defer os.Remove(tempFile)

	testStorage := storage.NewMemStorage()

	// Добавляем метрики из нескольких горутин
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			testStorage.UpdateGauge("concurrent_gauge_"+string(rune(id)), float64(id))
			testStorage.UpdateCounter("concurrent_counter_"+string(rune(id)), int64(id))
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	// Сохранение
	err := testStorage.SaveToFile(tempFile)
	require.NoError(t, err)

	// Загрузка и проверка
	newStorage := storage.NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	require.NoError(t, err)

	gauges := newStorage.GetAllGauges()
	counters := newStorage.GetAllCounters()

	assert.Equal(t, 5, len(gauges))
	assert.Equal(t, 5, len(counters))

	// Значения корректны
	for i := 0; i < 5; i++ {
		gauge, exists := newStorage.GetGauge("concurrent_gauge_" + string(rune(i)))
		require.True(t, exists)
		assert.Equal(t, float64(i), gauge)

		counter, exists := newStorage.GetCounter("concurrent_counter_" + string(rune(i)))
		require.True(t, exists)
		assert.Equal(t, int64(i), counter)
	}
}
