package grpc

import (
	"context"
	"fmt"
	"log"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
)

// MetricsServer реализует gRPC сервер для метрик
type MetricsServer struct {
	// Встраиваем UnimplementedMetricsServiceServer для совместимости
	pb.UnimplementedMetricsServiceServer

	storage storage.Storage
	key     string // ключ для проверки хешей
}

// NewMetricsServer создает новый gRPC сервер для метрик
func NewMetricsServer(storage storage.Storage, key string) *MetricsServer {
	return &MetricsServer{
		storage: storage,
		key:     key,
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
	if s.key != "" && !s.verifyMetricHash(req.Metric) {
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

		// Возвращаем обновленную метрику
		response := &pb.UpdateMetricResponse{
			Metric: &pb.Metric{
				Id:    req.Metric.Id,
				Type:  req.Metric.Type,
				Value: req.Metric.Value,
			},
		}
		s.addHashToMetric(response.Metric)
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
			Metric: &pb.Metric{
				Id:    req.Metric.Id,
				Type:  req.Metric.Type,
				Delta: &value,
			},
		}
		s.addHashToMetric(response.Metric)
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

		metric := &pb.Metric{
			Id:    req.Id,
			Type:  req.Type,
			Value: &value,
		}
		s.addHashToMetric(metric)

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

		metric := &pb.Metric{
			Id:    req.Id,
			Type:  req.Type,
			Delta: &value,
		}
		s.addHashToMetric(metric)

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
	if s.key != "" {
		for i, metric := range req.Metrics {
			if !s.verifyMetricHash(metric) {
				return &pb.UpdateMetricsResponse{
					Error: fmt.Sprintf("hash verification failed for metric %d: %s", i, metric.Id),
				}, nil
			}
		}
	}

	// Конвертируем protobuf метрики в внутренний формат
	metricsModels := make([]models.Metrics, 0, len(req.Metrics))
	for _, pbMetric := range req.Metrics {
		model := models.Metrics{
			ID:    pbMetric.Id,
			MType: pbMetric.Type,
		}

		switch pbMetric.Type {
		case "gauge":
			if pbMetric.Value == nil {
				return &pb.UpdateMetricsResponse{
					Error: fmt.Sprintf("value is required for gauge metric: %s", pbMetric.Id),
				}, nil
			}
			model.Value = pbMetric.Value
		case "counter":
			if pbMetric.Delta == nil {
				return &pb.UpdateMetricsResponse{
					Error: fmt.Sprintf("delta is required for counter metric: %s", pbMetric.Id),
				}, nil
			}
			model.Delta = pbMetric.Delta
		default:
			return &pb.UpdateMetricsResponse{
				Error: fmt.Sprintf("unknown metric type: %s for metric %s", pbMetric.Type, pbMetric.Id),
			}, nil
		}

		metricsModels = append(metricsModels, model)
	}

	// Обновляем все метрики в batch
	if err := s.storage.UpdateBatch(metricsModels); err != nil {
		log.Printf("Failed to update batch: %v", err)
		return &pb.UpdateMetricsResponse{
			Error: fmt.Sprintf("failed to update metrics: %v", err),
		}, nil
	}

	return &pb.UpdateMetricsResponse{}, nil
}

// GetAllMetrics получает все метрики в системе
func (s *MetricsServer) GetAllMetrics(ctx context.Context, req *pb.GetAllMetricsRequest) (*pb.GetAllMetricsResponse, error) {
	var pbMetrics []*pb.Metric

	// Получаем все gauge метрики
	gauges := s.storage.GetAllGauges()
	for name, value := range gauges {
		metric := &pb.Metric{
			Id:    name,
			Type:  "gauge",
			Value: &value,
		}
		s.addHashToMetric(metric)
		pbMetrics = append(pbMetrics, metric)
	}

	// Получаем все counter метрики
	counters := s.storage.GetAllCounters()
	for name, value := range counters {
		metric := &pb.Metric{
			Id:    name,
			Type:  "counter",
			Delta: &value,
		}
		s.addHashToMetric(metric)
		pbMetrics = append(pbMetrics, metric)
	}

	return &pb.GetAllMetricsResponse{
		Metrics: pbMetrics,
	}, nil
}

// Ping проверяет здоровье сервиса и доступность хранилища
func (s *MetricsServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	if !s.storage.IsAvailable() {
		return &pb.PingResponse{
			Ok:    false,
			Error: "storage is not available",
		}, nil
	}

	if err := s.storage.Ping(); err != nil {
		return &pb.PingResponse{
			Ok:    false,
			Error: fmt.Sprintf("storage ping failed: %v", err),
		}, nil
	}

	return &pb.PingResponse{
		Ok: true,
	}, nil
}

// addHashToMetric добавляет хеш к метрике если ключ задан
func (s *MetricsServer) addHashToMetric(metric *pb.Metric) {
	if s.key == "" {
		return
	}

	var data string
	switch metric.Type {
	case "counter":
		if metric.Delta != nil {
			data = fmt.Sprintf("%s:%s:%d", metric.Id, metric.Type, *metric.Delta)
		}
	case "gauge":
		if metric.Value != nil {
			data = fmt.Sprintf("%s:%s:%f", metric.Id, metric.Type, *metric.Value)
		}
	}

	if data != "" {
		metric.Hash = utils.CalculateHash([]byte(data), s.key)
	}
}

// verifyMetricHash проверяет хеш метрики
func (s *MetricsServer) verifyMetricHash(metric *pb.Metric) bool {
	if s.key == "" {
		return true
	}

	if metric.Hash == "" {
		log.Printf("No hash provided for metric: %s, type: %s", metric.Id, metric.Type)
		return false
	}

	var data string
	switch metric.Type {
	case "counter":
		if metric.Delta == nil {
			log.Printf("Counter metric %s has nil delta", metric.Id)
			return false
		}
		data = fmt.Sprintf("%s:%s:%d", metric.Id, metric.Type, *metric.Delta)
	case "gauge":
		if metric.Value == nil {
			log.Printf("Gauge metric %s has nil value", metric.Id)
			return false
		}
		data = fmt.Sprintf("%s:%s:%f", metric.Id, metric.Type, *metric.Value)
	default:
		log.Printf("Unknown metric type: %s for metric %s", metric.Type, metric.Id)
		return false
	}

	expectedHash := utils.CalculateHash([]byte(data), s.key)
	if metric.Hash != expectedHash {
		log.Printf("Hash mismatch for metric %s: expected %s, got %s", metric.Id, expectedHash, metric.Hash)
		return false
	}

	return true
}
