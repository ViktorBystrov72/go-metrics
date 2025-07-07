package storage

import (
	"testing"
)

func TestStorage_GetGauge_Error(t *testing.T) {
	storage := NewMemStorage()

	_, err := storage.GetGauge("nonexistent")
	if err == nil {
		t.Error("Expected error when getting nonexistent gauge metric")
	}

	storage.UpdateGauge("test_gauge", 123.45)
	value, err := storage.GetGauge("test_gauge")
	if err != nil {
		t.Errorf("Unexpected error when getting existing gauge metric: %v", err)
	}
	if value != 123.45 {
		t.Errorf("Expected value 123.45, got %f", value)
	}
}

func TestStorage_GetCounter_Error(t *testing.T) {
	storage := NewMemStorage()

	_, err := storage.GetCounter("nonexistent")
	if err == nil {
		t.Error("Expected error when getting nonexistent counter metric")
	}

	storage.UpdateCounter("test_counter", 42)
	value, err := storage.GetCounter("test_counter")
	if err != nil {
		t.Errorf("Unexpected error when getting existing counter metric: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected value 42, got %d", value)
	}
}
