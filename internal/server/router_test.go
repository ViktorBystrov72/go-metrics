package server

import (
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

func TestNewRouter(t *testing.T) {
	storage := storage.NewMemStorage()
	router := NewRouter(storage, "", "")
	if router == nil {
		t.Error("NewRouter() вернул nil")
	}
}

func TestRouter_WithLogging(t *testing.T) {
	storage := storage.NewMemStorage()
	router := NewRouter(storage, "", "")
	if router == nil {
		t.Error("NewRouter() вернул nil")
	}
}

func TestRouter_GetRouter(t *testing.T) {
	storage := storage.NewMemStorage()
	router := NewRouter(storage, "", "")
	chiRouter := router.GetRouter()
	if chiRouter == nil {
		t.Error("GetRouter() вернул nil")
	}
}
