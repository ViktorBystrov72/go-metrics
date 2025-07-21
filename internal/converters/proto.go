package converters

import (
	"github.com/ViktorBystrov72/go-metrics/internal/models"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
)

// ModelToProto конвертирует внутреннюю модель в protobuf формат
func ModelToProto(metric models.Metrics) *pb.Metric {
	pbMetric := &pb.Metric{
		Id:   metric.ID,
		Type: metric.MType,
		Hash: metric.Hash,
	}

	switch metric.MType {
	case "gauge":
		if metric.Value != nil {
			pbMetric.Value = metric.Value
		}
	case "counter":
		if metric.Delta != nil {
			pbMetric.Delta = metric.Delta
		}
	}

	return pbMetric
}

// ProtoToModel конвертирует protobuf формат во внутреннюю модель
func ProtoToModel(pbMetric *pb.Metric) models.Metrics {
	metric := models.Metrics{
		ID:    pbMetric.Id,
		MType: pbMetric.Type,
		Hash:  pbMetric.Hash,
	}

	switch pbMetric.Type {
	case "gauge":
		if pbMetric.Value != nil {
			metric.Value = pbMetric.Value
		}
	case "counter":
		if pbMetric.Delta != nil {
			metric.Delta = pbMetric.Delta
		}
	}

	return metric
}

// CreateGaugeProto создает protobuf метрику типа gauge
func CreateGaugeProto(id string, value float64) *pb.Metric {
	return &pb.Metric{
		Id:    id,
		Type:  "gauge",
		Value: &value,
	}
}

// CreateCounterProto создает protobuf метрику типа counter
func CreateCounterProto(id string, delta int64) *pb.Metric {
	return &pb.Metric{
		Id:    id,
		Type:  "counter",
		Delta: &delta,
	}
}

// ModelsToProtos конвертирует slice внутренних моделей в slice protobuf моделей
func ModelsToProtos(metrics []models.Metrics) []*pb.Metric {
	protos := make([]*pb.Metric, 0, len(metrics))
	for _, metric := range metrics {
		protos = append(protos, ModelToProto(metric))
	}
	return protos
}

// ProtosToModels конвертирует slice protobuf моделей в slice внутренних моделей
func ProtosToModels(pbMetrics []*pb.Metric) []models.Metrics {
	models := make([]models.Metrics, 0, len(pbMetrics))
	for _, pbMetric := range pbMetrics {
		models = append(models, ProtoToModel(pbMetric))
	}
	return models
}

// ValidateProtoMetric проверяет корректность protobuf метрики
func ValidateProtoMetric(pbMetric *pb.Metric) error {
	if pbMetric.Id == "" {
		return ErrMissingID
	}

	switch pbMetric.Type {
	case "gauge":
		if pbMetric.Value == nil {
			return ErrMissingValue
		}
	case "counter":
		if pbMetric.Delta == nil {
			return ErrMissingDelta
		}
	default:
		return ErrUnknownType
	}

	return nil
}
