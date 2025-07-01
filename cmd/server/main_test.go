package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
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
