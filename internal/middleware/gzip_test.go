package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGzipMiddlewareWithUnsupportedContentType тестирует middleware с неподдерживаемым типом контента.
func TestGzipMiddlewareWithUnsupportedContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("test response"))
	})

	middleware := GzipMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestGzipMiddlewareWithoutAcceptEncoding тестирует middleware без заголовка Accept-Encoding.
func TestGzipMiddlewareWithoutAcceptEncoding(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"test": "response"}`))
	})

	middleware := GzipMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}

	// Проверяем, что ответ не сжат
	contentEncoding := w.Header().Get("Content-Encoding")
	if contentEncoding == "gzip" {
		t.Error("Ответ не должен быть сжат без Accept-Encoding")
	}
}

// TestGzipMiddlewareWithEmptyBody тестирует middleware с пустым телом ответа.
func TestGzipMiddlewareWithEmptyBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Пустое тело
	})

	middleware := GzipMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

// TestGzipMiddlewareWithLargeResponse тестирует middleware с большим ответом.
func TestGzipMiddlewareWithLargeResponse(t *testing.T) {
	// Создаем большой ответ
	largeResponse := bytes.Repeat([]byte("test data "), 1000)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(largeResponse)
	})

	middleware := GzipMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}

	// Проверяем, что ответ сжат
	contentEncoding := w.Header().Get("Content-Encoding")
	if contentEncoding != "gzip" {
		t.Error("Большой ответ должен быть сжат")
	}

	// Проверяем, что сжатые данные можно распаковать
	body := w.Body.Bytes()
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Не удалось создать gzip reader: %v", err)
	}
	defer reader.Close()

	uncompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Не удалось распаковать данные: %v", err)
	}

	if !bytes.Equal(uncompressed, largeResponse) {
		t.Error("Распакованные данные не совпадают с оригинальными")
	}
}

// TestGzipMiddlewareWithMultipleEncodings тестирует middleware с несколькими кодировками.
func TestGzipMiddlewareWithMultipleEncodings(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"test": "response"}`))
	})

	middleware := GzipMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "deflate, gzip, br")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}

	// Проверяем, что ответ сжат
	contentEncoding := w.Header().Get("Content-Encoding")
	if contentEncoding != "gzip" {
		t.Error("Ответ должен быть сжат при поддержке gzip")
	}
}

// TestGzipMiddlewareWithGzipOnly тестирует middleware только с gzip.
func TestGzipMiddlewareWithGzipOnly(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"test": "response"}`))
	})

	middleware := GzipMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}

	// Проверяем, что ответ сжат
	contentEncoding := w.Header().Get("Content-Encoding")
	if contentEncoding != "gzip" {
		t.Error("Ответ должен быть сжат при поддержке gzip")
	}
}
