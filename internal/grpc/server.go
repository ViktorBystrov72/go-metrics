package grpc

import (
	"context"
	"fmt"

	"github.com/ViktorBystrov72/go-metrics/internal/converters"
	"github.com/ViktorBystrov72/go-metrics/internal/hashservice"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
)

// MetricsServer реализует gRPC сервер для метрик
type MetricsServer struct {
	// Встраиваем UnimplementedMetricsServiceServer для совместимости
	pb.UnimplementedMetricsServiceServer

	storage storage.Storage
	hasher  *hashservice.ProtoHasher
}

// NewMetricsServer создает новый gRPC сервер для метрик
func NewMetricsServer(storage storage.Storage, key string) *MetricsServer {
	return &MetricsServer{
		storage: storage,
		hasher:  hashservice.NewProtoHasher(key),
	}
}

// UpdateMetric обновляет одну метрику
func (s *MetricsServer) UpdateMetric(ctx context.Context, req *pb.UpdateMetricRequest) (*pb.UpdateMetricResponse, error) {
	if req.Metric == nil {
		return &pb.UpdateMetricResponse{
			Error: "metric is required",
		}, nil
	}

	// Проверяем хеш если ключ задан
	if s.hasher.IsEnabled() && !s.hasher.VerifyHash(req.Metric) {
		return &pb.UpdateMetricResponse{
			Error: "hash verification failed",
		}, nil
	}

	// Обновляем метрику
	switch req.Metric.Type {
	case "gauge":
		if req.Metric.Value == nil {
			return &pb.UpdateMetricResponse{
				Error: "value is required for gauge metric",
			}, nil
		}
		s.storage.UpdateGauge(req.Metric.Id, *req.Metric.Value)

		// Возвращаем обновленную метрику используя конвертер
		response := &pb.UpdateMetricResponse{
			Metric: converters.CreateGaugeProto(req.Metric.Id, *req.Metric.Value),
		}
		s.hasher.AddHash(response.Metric)
		return response, nil

	case "counter":
		if req.Metric.Delta == nil {
			return &pb.UpdateMetricResponse{
				Error: "delta is required for counter metric",
			}, nil
		}
		s.storage.UpdateCounter(req.Metric.Id, *req.Metric.Delta)

		// Получаем актуальное значение после обновления
		value, err := s.storage.GetCounter(req.Metric.Id)
		if err != nil {
			return &pb.UpdateMetricResponse{
				Error: fmt.Sprintf("failed to get updated counter value: %v", err),
			}, nil
		}

		response := &pb.UpdateMetricResponse{
			Metric: converters.CreateCounterProto(req.Metric.Id, value),
		}
		s.hasher.AddHash(response.Metric)
		return response, nil

	default:
		return &pb.UpdateMetricResponse{
			Error: fmt.Sprintf("unknown metric type: %s", req.Metric.Type),
		}, nil
	}
}

// GetMetric получает значение одной метрики
func (s *MetricsServer) GetMetric(ctx context.Context, req *pb.GetMetricRequest) (*pb.GetMetricResponse, error) {
	switch req.Type {
	case "gauge":
		value, err := s.storage.GetGauge(req.Id)
		if err != nil {
			return &pb.GetMetricResponse{
				Error: fmt.Sprintf("metric not found: %s", req.Id),
			}, nil
		}

		metric := converters.CreateGaugeProto(req.Id, value)
		s.hasher.AddHash(metric)

		return &pb.GetMetricResponse{
			Metric: metric,
		}, nil

	case "counter":
		value, err := s.storage.GetCounter(req.Id)
		if err != nil {
			return &pb.GetMetricResponse{
				Error: fmt.Sprintf("metric not found: %s", req.Id),
			}, nil
		}

		metric := converters.CreateCounterProto(req.Id, value)
		s.hasher.AddHash(metric)

		return &pb.GetMetricResponse{
			Metric: metric,
		}, nil

	default:
		return &pb.GetMetricResponse{
			Error: fmt.Sprintf("unknown metric type: %s", req.Type),
		}, nil
	}
}

// UpdateMetrics обновляет множество метрик в одном запросе (batch)
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if len(req.Metrics) == 0 {
		return &pb.UpdateMetricsResponse{
			Error: "no metrics to update",
		}, nil
	}

	// Проверяем хеши всех метрик если ключ задан
	if s.hasher.IsEnabled() {
		if err := s.hasher.VerifyHashes(req.Metrics); err != nil {
			return &pb.UpdateMetricsResponse{
				Error: err.Error(),
			}, nil
		}
	}

	// Валидируем и конвертируем protobuf метрики в внутренний формат используя общий конвертер
	for _, pbMetric := range req.Metrics {
		if err := converters.ValidateProtoMetric(pbMetric); err != nil {
			return &pb.UpdateMetricsResponse{
				Error: fmt.Sprintf("validation failed for metric %s: %v", pbMetric.Id, err),
			}, nil
		}
	}

	// Конвертируем protobuf метрики в модели
	metricsModels := converters.ProtosToModels(req.Metrics)

	// Обновляем все метрики в batch
	if err := s.storage.UpdateBatch(metricsModels); err != nil {
		return &pb.UpdateMetricsResponse{
			Error: fmt.Sprintf("failed to update metrics: %v", err),
		}, nil
	}

	return &pb.UpdateMetricsResponse{}, nil
}

// GetAllMetrics получает все метрики
func (s *MetricsServer) GetAllMetrics(ctx context.Context, req *pb.GetAllMetricsRequest) (*pb.GetAllMetricsResponse, error) {
	// Получаем все gauge метрики
	gauges := s.storage.GetAllGauges()
	gaugeProtos := make([]*pb.Metric, 0, len(gauges))
	for name, value := range gauges {
		metric := converters.CreateGaugeProto(name, value)
		s.hasher.AddHash(metric)
		gaugeProtos = append(gaugeProtos, metric)
	}

	// Получаем все counter метрики
	counters := s.storage.GetAllCounters()
	counterProtos := make([]*pb.Metric, 0, len(counters))
	for name, value := range counters {
		metric := converters.CreateCounterProto(name, value)
		s.hasher.AddHash(metric)
		counterProtos = append(counterProtos, metric)
	}

	// Объединяем все метрики
	allMetrics := append(gaugeProtos, counterProtos...)

	return &pb.GetAllMetricsResponse{
		Metrics: allMetrics,
	}, nil
}

// Ping проверяет здоровье сервиса
func (s *MetricsServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	// Проверяем доступность хранилища
	if err := s.storage.Ping(); err != nil {
		return &pb.PingResponse{
			Ok:    false,
			Error: fmt.Sprintf("storage unavailable: %v", err),
		}, nil
	}

	return &pb.PingResponse{
		Ok: true,
	}, nil
}
