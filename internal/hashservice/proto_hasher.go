package hashservice

import (
	"fmt"

	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
)

// ProtoHasher сервис для работы с хешами protobuf метрик
type ProtoHasher struct {
	key string // ключ для подписи
}

// NewProtoHasher создает новый сервис хеширования protobuf метрик
func NewProtoHasher(key string) *ProtoHasher {
	return &ProtoHasher{
		key: key,
	}
}

// AddHash добавляет хеш к protobuf метрике
func (h *ProtoHasher) AddHash(metric *pb.Metric) {
	if h.key == "" {
		return
	}

	data := h.buildHashData(metric)
	if data != "" {
		metric.Hash = utils.CalculateHash([]byte(data), h.key)
	}
}

// VerifyHash проверяет хеш protobuf метрики
func (h *ProtoHasher) VerifyHash(metric *pb.Metric) bool {
	if h.key == "" {
		return true // если ключ не задан, считаем проверку пройденной
	}

	data := h.buildHashData(metric)
	if data == "" {
		return false
	}

	expectedHash := utils.CalculateHash([]byte(data), h.key)
	return expectedHash == metric.Hash
}

// AddHashes добавляет хеши к массиву protobuf метрик
func (h *ProtoHasher) AddHashes(metrics []*pb.Metric) {
	for _, metric := range metrics {
		h.AddHash(metric)
	}
}

// VerifyHashes проверяет хеши массива protobuf метрик
func (h *ProtoHasher) VerifyHashes(metrics []*pb.Metric) error {
	for i, metric := range metrics {
		if !h.VerifyHash(metric) {
			return fmt.Errorf("hash verification failed for metric %d: %s", i, metric.Id)
		}
	}
	return nil
}

// IsEnabled возвращает true если хеширование включено
func (h *ProtoHasher) IsEnabled() bool {
	return h.key != ""
}

// buildHashData создает строку для хеширования из protobuf метрики
func (h *ProtoHasher) buildHashData(metric *pb.Metric) string {
	switch metric.Type {
	case "counter":
		if metric.Delta != nil {
			return fmt.Sprintf("%s:%s:%d", metric.Id, metric.Type, *metric.Delta)
		}
	case "gauge":
		if metric.Value != nil {
			return fmt.Sprintf("%s:%s:%f", metric.Id, metric.Type, *metric.Value)
		}
	}
	return ""
}
