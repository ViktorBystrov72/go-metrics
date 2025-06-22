package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	testStorage := storage.NewMemStorage()
	server := NewServer(testStorage)

	r := chi.NewRouter()
	r.Route("/update", func(r chi.Router) {
		r.Post("/{type}/{name}/{value}", server.updateHandler)
	})
	r.Route("/value", func(r chi.Router) {
		r.Get("/{type}/{name}", server.valueHandler)
	})
	r.Get("/", server.indexHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("Update Gauge", func(t *testing.T) {
		url := fmt.Sprintf("%s/update/gauge/testGauge/123.45", ts.URL)
		resp, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte{}))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Update Counter", func(t *testing.T) {
		url := fmt.Sprintf("%s/update/counter/testCounter/10", ts.URL)
		resp, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte{}))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Get Gauge Value", func(t *testing.T) {
		updateURL := fmt.Sprintf("%s/update/gauge/testGauge/123.45", ts.URL)
		resp, err := http.Post(updateURL, "text/plain", bytes.NewBuffer([]byte{}))
		require.NoError(t, err)
		defer resp.Body.Close()

		getURL := fmt.Sprintf("%s/value/gauge/testGauge", ts.URL)
		resp, err = http.Get(getURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Get Counter Value", func(t *testing.T) {
		updateURL := fmt.Sprintf("%s/update/counter/testCounter/5", ts.URL)
		resp, err := http.Post(updateURL, "text/plain", bytes.NewBuffer([]byte{}))
		require.NoError(t, err)
		defer resp.Body.Close()

		getURL := fmt.Sprintf("%s/value/counter/testCounter", ts.URL)
		resp, err = http.Get(getURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Get Non-existent Metric", func(t *testing.T) {
		getURL := fmt.Sprintf("%s/value/gauge/nonExistent", ts.URL)
		resp, err := http.Get(getURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Get Index Page", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))
	})

	t.Run("Invalid Metric Type", func(t *testing.T) {
		url := fmt.Sprintf("%s/update/invalid/testMetric/123", ts.URL)
		resp, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte{}))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Invalid Value", func(t *testing.T) {
		url := fmt.Sprintf("%s/update/gauge/testMetric/invalid", ts.URL)
		resp, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte{}))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
