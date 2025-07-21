package converters

import (
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
)

func TestModelToProto(t *testing.T) {
	tests := []struct {
		name     string
		metric   models.Metrics
		expected *pb.Metric
	}{
		{
			name: "gauge metric",
			metric: models.Metrics{
				ID:    "test_gauge",
				MType: "gauge",
				Value: func() *float64 { v := 123.45; return &v }(),
				Hash:  "test_hash",
			},
			expected: &pb.Metric{
				Id:    "test_gauge",
				Type:  "gauge",
				Value: func() *float64 { v := 123.45; return &v }(),
				Hash:  "test_hash",
			},
		},
		{
			name: "counter metric",
			metric: models.Metrics{
				ID:    "test_counter",
				MType: "counter",
				Delta: func() *int64 { v := int64(10); return &v }(),
				Hash:  "test_hash",
			},
			expected: &pb.Metric{
				Id:    "test_counter",
				Type:  "counter",
				Delta: func() *int64 { v := int64(10); return &v }(),
				Hash:  "test_hash",
			},
		},
		{
			name: "gauge metric without value",
			metric: models.Metrics{
				ID:    "test_gauge_empty",
				MType: "gauge",
				Value: nil,
			},
			expected: &pb.Metric{
				Id:   "test_gauge_empty",
				Type: "gauge",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ModelToProto(tt.metric)

			if result.Id != tt.expected.Id {
				t.Errorf("Expected ID %s, got %s", tt.expected.Id, result.Id)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Expected Type %s, got %s", tt.expected.Type, result.Type)
			}
			if result.Hash != tt.expected.Hash {
				t.Errorf("Expected Hash %s, got %s", tt.expected.Hash, result.Hash)
			}

			// Проверка Value
			if tt.expected.Value != nil {
				if result.Value == nil {
					t.Errorf("Expected Value %f, got nil", *tt.expected.Value)
				} else if *result.Value != *tt.expected.Value {
					t.Errorf("Expected Value %f, got %f", *tt.expected.Value, *result.Value)
				}
			} else if result.Value != nil {
				t.Errorf("Expected Value nil, got %f", *result.Value)
			}

			// Проверка Delta
			if tt.expected.Delta != nil {
				if result.Delta == nil {
					t.Errorf("Expected Delta %d, got nil", *tt.expected.Delta)
				} else if *result.Delta != *tt.expected.Delta {
					t.Errorf("Expected Delta %d, got %d", *tt.expected.Delta, *result.Delta)
				}
			} else if result.Delta != nil {
				t.Errorf("Expected Delta nil, got %d", *result.Delta)
			}
		})
	}
}

func TestProtoToModel(t *testing.T) {
	tests := []struct {
		name     string
		pbMetric *pb.Metric
		expected models.Metrics
	}{
		{
			name: "gauge metric",
			pbMetric: &pb.Metric{
				Id:    "test_gauge",
				Type:  "gauge",
				Value: func() *float64 { v := 123.45; return &v }(),
				Hash:  "test_hash",
			},
			expected: models.Metrics{
				ID:    "test_gauge",
				MType: "gauge",
				Value: func() *float64 { v := 123.45; return &v }(),
				Hash:  "test_hash",
			},
		},
		{
			name: "counter metric",
			pbMetric: &pb.Metric{
				Id:    "test_counter",
				Type:  "counter",
				Delta: func() *int64 { v := int64(10); return &v }(),
				Hash:  "test_hash",
			},
			expected: models.Metrics{
				ID:    "test_counter",
				MType: "counter",
				Delta: func() *int64 { v := int64(10); return &v }(),
				Hash:  "test_hash",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProtoToModel(tt.pbMetric)

			if result.ID != tt.expected.ID {
				t.Errorf("Expected ID %s, got %s", tt.expected.ID, result.ID)
			}
			if result.MType != tt.expected.MType {
				t.Errorf("Expected MType %s, got %s", tt.expected.MType, result.MType)
			}
			if result.Hash != tt.expected.Hash {
				t.Errorf("Expected Hash %s, got %s", tt.expected.Hash, result.Hash)
			}

			// Проверка Value
			if tt.expected.Value != nil {
				if result.Value == nil {
					t.Errorf("Expected Value %f, got nil", *tt.expected.Value)
				} else if *result.Value != *tt.expected.Value {
					t.Errorf("Expected Value %f, got %f", *tt.expected.Value, *result.Value)
				}
			}

			// Проверка Delta
			if tt.expected.Delta != nil {
				if result.Delta == nil {
					t.Errorf("Expected Delta %d, got nil", *tt.expected.Delta)
				} else if *result.Delta != *tt.expected.Delta {
					t.Errorf("Expected Delta %d, got %d", *tt.expected.Delta, *result.Delta)
				}
			}
		})
	}
}

func TestCreateGaugeProto(t *testing.T) {
	id := "test_gauge"
	value := 123.45

	result := CreateGaugeProto(id, value)

	if result.Id != id {
		t.Errorf("Expected ID %s, got %s", id, result.Id)
	}
	if result.Type != "gauge" {
		t.Errorf("Expected Type gauge, got %s", result.Type)
	}
	if result.Value == nil {
		t.Errorf("Expected Value %f, got nil", value)
	} else if *result.Value != value {
		t.Errorf("Expected Value %f, got %f", value, *result.Value)
	}
}

func TestCreateCounterProto(t *testing.T) {
	id := "test_counter"
	delta := int64(10)

	result := CreateCounterProto(id, delta)

	if result.Id != id {
		t.Errorf("Expected ID %s, got %s", id, result.Id)
	}
	if result.Type != "counter" {
		t.Errorf("Expected Type counter, got %s", result.Type)
	}
	if result.Delta == nil {
		t.Errorf("Expected Delta %d, got nil", delta)
	} else if *result.Delta != delta {
		t.Errorf("Expected Delta %d, got %d", delta, *result.Delta)
	}
}

func TestValidateProtoMetric(t *testing.T) {
	tests := []struct {
		name      string
		pbMetric  *pb.Metric
		expectErr error
	}{
		{
			name: "valid gauge",
			pbMetric: &pb.Metric{
				Id:    "test",
				Type:  "gauge",
				Value: func() *float64 { v := 123.0; return &v }(),
			},
			expectErr: nil,
		},
		{
			name: "valid counter",
			pbMetric: &pb.Metric{
				Id:    "test",
				Type:  "counter",
				Delta: func() *int64 { v := int64(10); return &v }(),
			},
			expectErr: nil,
		},
		{
			name: "missing ID",
			pbMetric: &pb.Metric{
				Type:  "gauge",
				Value: func() *float64 { v := 123.0; return &v }(),
			},
			expectErr: ErrMissingID,
		},
		{
			name: "gauge without value",
			pbMetric: &pb.Metric{
				Id:   "test",
				Type: "gauge",
			},
			expectErr: ErrMissingValue,
		},
		{
			name: "counter without delta",
			pbMetric: &pb.Metric{
				Id:   "test",
				Type: "counter",
			},
			expectErr: ErrMissingDelta,
		},
		{
			name: "unknown type",
			pbMetric: &pb.Metric{
				Id:   "test",
				Type: "unknown",
			},
			expectErr: ErrUnknownType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProtoMetric(tt.pbMetric)

			if tt.expectErr == nil && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if tt.expectErr != nil && err != tt.expectErr {
				t.Errorf("Expected error %v, got %v", tt.expectErr, err)
			}
		})
	}
}
