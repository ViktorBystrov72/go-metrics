package hashservice

import (
	"testing"

	pb "github.com/ViktorBystrov72/go-metrics/proto"
)

func TestNewProtoHasher(t *testing.T) {
	hasher := NewProtoHasher("test-key")
	if hasher.key != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", hasher.key)
	}
}

func TestProtoHasher_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "with key",
			key:      "test-key",
			expected: true,
		},
		{
			name:     "without key",
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewProtoHasher(tt.key)
			if hasher.IsEnabled() != tt.expected {
				t.Errorf("Expected IsEnabled() = %v, got %v", tt.expected, hasher.IsEnabled())
			}
		})
	}
}

func TestProtoHasher_AddHash(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		metric     *pb.Metric
		expectHash bool
	}{
		{
			name: "gauge metric with key",
			key:  "test-key",
			metric: &pb.Metric{
				Id:    "test_gauge",
				Type:  "gauge",
				Value: func() *float64 { v := 123.45; return &v }(),
			},
			expectHash: true,
		},
		{
			name: "counter metric with key",
			key:  "test-key",
			metric: &pb.Metric{
				Id:    "test_counter",
				Type:  "counter",
				Delta: func() *int64 { v := int64(10); return &v }(),
			},
			expectHash: true,
		},
		{
			name: "gauge metric without key",
			key:  "",
			metric: &pb.Metric{
				Id:    "test_gauge",
				Type:  "gauge",
				Value: func() *float64 { v := 123.45; return &v }(),
			},
			expectHash: false,
		},
		{
			name: "gauge metric without value",
			key:  "test-key",
			metric: &pb.Metric{
				Id:   "test_gauge",
				Type: "gauge",
			},
			expectHash: false,
		},
		{
			name: "counter metric without delta",
			key:  "test-key",
			metric: &pb.Metric{
				Id:   "test_counter",
				Type: "counter",
			},
			expectHash: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewProtoHasher(tt.key)
			originalHash := tt.metric.Hash

			hasher.AddHash(tt.metric)

			if tt.expectHash {
				if tt.metric.Hash == "" {
					t.Errorf("Expected hash to be added, but got empty string")
				}
				if tt.metric.Hash == originalHash {
					t.Errorf("Expected hash to be different from original")
				}
			} else {
				if tt.metric.Hash != originalHash {
					t.Errorf("Expected hash to remain unchanged")
				}
			}
		})
	}
}

func TestProtoHasher_VerifyHash(t *testing.T) {
	hasher := NewProtoHasher("test-key")

	// Создаем метрику и добавляем к ней хеш
	metric := &pb.Metric{
		Id:    "test_gauge",
		Type:  "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
	}
	hasher.AddHash(metric)

	// Проверяем что хеш верифицируется
	if !hasher.VerifyHash(metric) {
		t.Errorf("Expected hash verification to succeed")
	}

	// Изменяем значение и проверяем что хеш не верифицируется
	newValue := 999.99
	metric.Value = &newValue
	if hasher.VerifyHash(metric) {
		t.Errorf("Expected hash verification to fail after value change")
	}
}

func TestProtoHasher_VerifyHash_NoKey(t *testing.T) {
	hasher := NewProtoHasher("")

	metric := &pb.Metric{
		Id:    "test_gauge",
		Type:  "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
		Hash:  "some-hash",
	}

	// Без ключа верификация должна всегда проходить
	if !hasher.VerifyHash(metric) {
		t.Errorf("Expected hash verification to succeed when no key is set")
	}
}

func TestProtoHasher_AddHashes(t *testing.T) {
	hasher := NewProtoHasher("test-key")

	metrics := []*pb.Metric{
		{
			Id:    "gauge1",
			Type:  "gauge",
			Value: func() *float64 { v := 123.45; return &v }(),
		},
		{
			Id:    "counter1",
			Type:  "counter",
			Delta: func() *int64 { v := int64(10); return &v }(),
		},
	}

	hasher.AddHashes(metrics)

	for i, metric := range metrics {
		if metric.Hash == "" {
			t.Errorf("Expected hash to be added to metric %d", i)
		}
	}
}

func TestProtoHasher_VerifyHashes(t *testing.T) {
	hasher := NewProtoHasher("test-key")

	metrics := []*pb.Metric{
		{
			Id:    "gauge1",
			Type:  "gauge",
			Value: func() *float64 { v := 123.45; return &v }(),
		},
		{
			Id:    "counter1",
			Type:  "counter",
			Delta: func() *int64 { v := int64(10); return &v }(),
		},
	}

	// Добавляем хеши
	hasher.AddHashes(metrics)

	// Проверяем что все хеши корректны
	if err := hasher.VerifyHashes(metrics); err != nil {
		t.Errorf("Expected hash verification to succeed, got: %v", err)
	}

	// Портим один хеш
	metrics[0].Hash = "invalid-hash"

	// Проверяем что верификация падает
	if err := hasher.VerifyHashes(metrics); err == nil {
		t.Errorf("Expected hash verification to fail")
	}
}

func TestProtoHasher_buildHashData(t *testing.T) {
	hasher := NewProtoHasher("test-key")

	tests := []struct {
		name     string
		metric   *pb.Metric
		expected string
	}{
		{
			name: "gauge metric",
			metric: &pb.Metric{
				Id:    "test_gauge",
				Type:  "gauge",
				Value: func() *float64 { v := 123.45; return &v }(),
			},
			expected: "test_gauge:gauge:123.450000",
		},
		{
			name: "counter metric",
			metric: &pb.Metric{
				Id:    "test_counter",
				Type:  "counter",
				Delta: func() *int64 { v := int64(10); return &v }(),
			},
			expected: "test_counter:counter:10",
		},
		{
			name: "gauge without value",
			metric: &pb.Metric{
				Id:   "test_gauge",
				Type: "gauge",
			},
			expected: "",
		},
		{
			name: "counter without delta",
			metric: &pb.Metric{
				Id:   "test_counter",
				Type: "counter",
			},
			expected: "",
		},
		{
			name: "unknown type",
			metric: &pb.Metric{
				Id:   "test_unknown",
				Type: "unknown",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasher.buildHashData(tt.metric)
			if result != tt.expected {
				t.Errorf("Expected buildHashData() = '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
