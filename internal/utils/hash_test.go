package utils

import (
	"testing"
)

func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		key      string
		expected string
	}{
		{
			name:     "empty key",
			data:     []byte("test data"),
			key:      "",
			expected: "",
		},
		{
			name:     "with key",
			data:     []byte("test data"),
			key:      "test key",
			expected: "a7c4c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateHash(tt.data, tt.key)
			if tt.key == "" {
				if result != "" {
					t.Errorf("Expected empty string for empty key, got %s", result)
				}
			} else {
				if result == "" {
					t.Error("Expected non-empty hash for non-empty key")
				}
				if len(result) != 64 {
					t.Errorf("Expected hash length 64, got %d", len(result))
				}
			}
		})
	}
}

func TestVerifyHash(t *testing.T) {
	data := []byte("test data")
	key := "test key"
	hash := CalculateHash(data, key)

	tests := []struct {
		name     string
		data     []byte
		key      string
		hash     string
		expected bool
	}{
		{
			name:     "valid hash",
			data:     data,
			key:      key,
			hash:     hash,
			expected: true,
		},
		{
			name:     "invalid hash",
			data:     data,
			key:      key,
			hash:     "invalid hash",
			expected: false,
		},
		{
			name:     "empty key",
			data:     data,
			key:      "",
			hash:     hash,
			expected: true,
		},
		{
			name:     "empty hash",
			data:     data,
			key:      key,
			hash:     "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyHash(tt.data, tt.key, tt.hash)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
