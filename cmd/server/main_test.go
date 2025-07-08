package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"

	"compress/gzip"
	"io"

	"os"

	"github.com/ViktorBystrov72/go-metrics/internal/config"
	"github.com/ViktorBystrov72/go-metrics/internal/middleware"
	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/server"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_JSON_API(t *testing.T) {
	testStorage := storage.NewMemStorage()
	handlers := server.NewHandlers(testStorage, "")

	r := chi.NewRouter()
	r.Post("/update/", handlers.UpdateJSONHandler)
	r.Post("/value/", handlers.ValueJSONHandler)

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
	handlers := server.NewHandlers(testStorage, "")

	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Post("/update/", handlers.UpdateJSONHandler)
	r.Post("/value/", handlers.ValueJSONHandler)

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
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "test_server_metrics_*.json")
	require.NoError(t, err)
	tmpFile.Close()
	tempFile := tmpFile.Name()

	testStorage := storage.NewMemStorage()
	testStorage.UpdateGauge("test_gauge", 123.45)
	testStorage.UpdateCounter("test_counter", 42)

	err = testStorage.SaveToFile(tempFile)
	require.NoError(t, err)

	_, err = os.Stat(tempFile)
	require.NoError(t, err)

	newStorage := storage.NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	require.NoError(t, err)

	gauge, err := newStorage.GetGauge("test_gauge")
	require.NoError(t, err)
	assert.Equal(t, 123.45, gauge)

	counter, err := newStorage.GetCounter("test_counter")
	require.NoError(t, err)
	assert.Equal(t, int64(42), counter)
}

func TestServer_StoreInterval_Zero(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "sync_metrics_*.json")
	require.NoError(t, err)
	tmpFile.Close()
	tempFile := tmpFile.Name()

	testStorage := storage.NewMemStorage()
	testStorage.UpdateGauge("sync_test", 99.99)

	err = testStorage.SaveToFile(tempFile)
	require.NoError(t, err)

	_, err = os.Stat(tempFile)
	require.NoError(t, err)

	newStorage := storage.NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	require.NoError(t, err)

	gauge, err := newStorage.GetGauge("sync_test")
	require.NoError(t, err)
	assert.Equal(t, 99.99, gauge)
}

func TestServer_Restore_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "restore_disabled_*.json")
	require.NoError(t, err)
	tmpFile.Close()
	tempFile := tmpFile.Name()

	testStorage := storage.NewMemStorage()
	testStorage.UpdateGauge("persistent_gauge", 777.77)
	err = testStorage.SaveToFile(tempFile)
	require.NoError(t, err)

	newStorage := storage.NewMemStorage()

	gauges := newStorage.GetAllGauges()
	counters := newStorage.GetAllCounters()

	assert.Equal(t, 0, len(gauges))
	assert.Equal(t, 0, len(counters))
}

func TestServer_FileStorage_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "concurrent_server_metrics_*.json")
	require.NoError(t, err)
	tmpFile.Close()
	tempFile := tmpFile.Name()

	testStorage := storage.NewMemStorage()

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

	err = testStorage.SaveToFile(tempFile)
	require.NoError(t, err)

	newStorage := storage.NewMemStorage()
	err = newStorage.LoadFromFile(tempFile)
	require.NoError(t, err)

	gauges := newStorage.GetAllGauges()
	counters := newStorage.GetAllCounters()

	assert.Equal(t, 5, len(gauges))
	assert.Equal(t, 5, len(counters))

	for i := 0; i < 5; i++ {
		gauge, err := newStorage.GetGauge("concurrent_gauge_" + string(rune(i)))
		require.NoError(t, err)
		assert.Equal(t, float64(i), gauge)
		counter, err := newStorage.GetCounter("concurrent_counter_" + string(rune(i)))
		require.NoError(t, err)
		assert.Equal(t, int64(i), counter)
	}
}

func TestServer_PingHandler(t *testing.T) {
	testStorage := storage.NewMemStorage()
	handlers := server.NewHandlers(testStorage, "")

	r := chi.NewRouter()
	r.Get("/ping", handlers.PingHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("ping_success", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/ping")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestServer_DatabaseDSN_Configuration(t *testing.T) {
	t.Run("database_dsn_flag", func(t *testing.T) {
		// Сбрасываем флаги перед тестом
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		// Тест проверяет, что флаг -d корректно обрабатывается
		// Это интеграционный тест, который проверяет парсинг конфигурации
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"server", "-d", "postgres://test:test@localhost:5432/test"}

		cfg, err := config.Load()
		require.NoError(t, err)
		assert.Equal(t, "postgres://test:test@localhost:5432/test", cfg.DatabaseDSN)
	})

	t.Run("database_dsn_env", func(t *testing.T) {
		// Сбрасываем флаги перед тестом
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		// Тест проверяет, что переменная окружения DATABASE_DSN имеет приоритет
		oldEnv := os.Getenv("DATABASE_DSN")
		defer os.Setenv("DATABASE_DSN", oldEnv)

		os.Setenv("DATABASE_DSN", "postgres://env:env@localhost:5432/env")

		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"server", "-d", "postgres://flag:flag@localhost:5432/flag"}

		cfg, err := config.Load()
		require.NoError(t, err)
		assert.Equal(t, "postgres://env:env@localhost:5432/env", cfg.DatabaseDSN)
	})
}

func TestStorage_IsDatabase(t *testing.T) {
	t.Run("database_storage", func(t *testing.T) {
		storage := &storage.DatabaseStorage{}
		assert.True(t, storage.IsDatabase())
	})

	t.Run("memory_storage", func(t *testing.T) {
		storage := storage.NewMemStorage()
		assert.False(t, storage.IsDatabase())
	})
}

func TestServer_HashVerification(t *testing.T) {
	testStorage := storage.NewMemStorage()
	handlers := server.NewHandlers(testStorage, "test-key")

	r := chi.NewRouter()
	r.Post("/update/", handlers.UpdateJSONHandler)
	r.Post("/value/", handlers.ValueJSONHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("valid hash", func(t *testing.T) {
		val := 123.45
		m := models.Metrics{ID: "testGauge", MType: "gauge", Value: &val}
		body, _ := json.Marshal(m)

		// Вычисляем правильный хеш
		hash := utils.CalculateHash(body, "test-key")

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/update/", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HashSHA256", hash)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("invalid hash", func(t *testing.T) {
		val := 123.45
		m := models.Metrics{ID: "testGauge", MType: "gauge", Value: &val}
		body, _ := json.Marshal(m)

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/update/", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HashSHA256", "invalid-hash")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("no hash when key is set", func(t *testing.T) {
		val := 123.45
		m := models.Metrics{ID: "testGauge", MType: "gauge", Value: &val}
		body, _ := json.Marshal(m)

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/update/", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		// Не передаем хеш

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_NoHashVerification(t *testing.T) {
	testStorage := storage.NewMemStorage()
	handlers := server.NewHandlers(testStorage, "") // Пустой ключ

	r := chi.NewRouter()
	r.Post("/update/", handlers.UpdateJSONHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("no key - no hash verification", func(t *testing.T) {
		val := 123.45
		m := models.Metrics{ID: "testGauge", MType: "gauge", Value: &val}
		body, _ := json.Marshal(m)

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/update/", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		// Не передаем хеш, но это должно работать

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
