package tests

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	grpcPkg "github.com/ViktorBystrov72/go-metrics/internal/grpc"
	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func bufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

// setupGRPCServer создает тестовый gRPC сервер
func setupGRPCServer(t *testing.T, trustedSubnet, key string) (*grpc.Server, *grpcPkg.MetricsServer, *bufconn.Listener) {
	storage := storage.NewMemStorage()
	metricsServer := grpcPkg.NewMetricsServer(storage, key)

	interceptor := grpcPkg.SetupServerInterceptors(trustedSubnet, "")
	s := grpc.NewServer(grpc.UnaryInterceptor(interceptor))

	pb.RegisterMetricsServiceServer(s, metricsServer)

	listener := bufconn.Listen(bufSize)

	go func() {
		if err := s.Serve(listener); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	return s, metricsServer, listener
}

// setupGRPCClient создает тестовый gRPC клиент
func setupGRPCClient(t *testing.T, listener *bufconn.Listener, key string) (*TestMetricsClient, func()) {
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer(listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcPkg.ClientIPInterceptor()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	// Создаем клиент через proto
	pbClient := pb.NewMetricsServiceClient(conn)

	// Создаем wrapper
	testClient := &TestMetricsClient{
		client: pbClient,
		key:    key,
	}

	return testClient, func() { conn.Close() }
}

// TestMetricsClient оборачивает gRPC клиент для тестирования
type TestMetricsClient struct {
	client pb.MetricsServiceClient
	key    string
}

func (c *TestMetricsClient) SendMetric(ctx context.Context, metric models.Metrics) error {
	pbMetric := &pb.Metric{
		Id:   metric.ID,
		Type: metric.MType,
		Hash: metric.Hash,
	}

	switch metric.MType {
	case "gauge":
		pbMetric.Value = metric.Value
	case "counter":
		pbMetric.Delta = metric.Delta
	}

	// Добавляем хеш если ключ задан
	c.addHashToMetric(pbMetric)

	req := &pb.UpdateMetricRequest{Metric: pbMetric}
	resp, err := c.client.UpdateMetric(ctx, req)
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return &GRPCError{Message: resp.Error}
	}

	return nil
}

func (c *TestMetricsClient) SendMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	pbMetrics := make([]*pb.Metric, 0, len(metrics))
	for _, metric := range metrics {
		pbMetric := &pb.Metric{
			Id:   metric.ID,
			Type: metric.MType,
			Hash: metric.Hash,
		}

		switch metric.MType {
		case "gauge":
			pbMetric.Value = metric.Value
		case "counter":
			pbMetric.Delta = metric.Delta
		}

		pbMetrics = append(pbMetrics, pbMetric)
	}

	req := &pb.UpdateMetricsRequest{Metrics: pbMetrics}
	resp, err := c.client.UpdateMetrics(ctx, req)
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return &GRPCError{Message: resp.Error}
	}

	return nil
}

func (c *TestMetricsClient) GetMetric(ctx context.Context, id, metricType string) (*models.Metrics, error) {
	req := &pb.GetMetricRequest{Id: id, Type: metricType}
	resp, err := c.client.GetMetric(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, &GRPCError{Message: resp.Error}
	}

	if resp.Metric == nil {
		return nil, &GRPCError{Message: "metric not found"}
	}

	metric := &models.Metrics{
		ID:    resp.Metric.Id,
		MType: resp.Metric.Type,
		Hash:  resp.Metric.Hash,
	}

	switch resp.Metric.Type {
	case "gauge":
		metric.Value = resp.Metric.Value
	case "counter":
		metric.Delta = resp.Metric.Delta
	}

	return metric, nil
}

func (c *TestMetricsClient) Ping(ctx context.Context) error {
	req := &pb.PingRequest{}
	resp, err := c.client.Ping(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return &GRPCError{Message: resp.Error}
	}

	return nil
}

type GRPCError struct {
	Message string
}

func (e *GRPCError) Error() string {
	return e.Message
}

// addHashToMetric добавляет хеш к метрике если ключ задан
func (c *TestMetricsClient) addHashToMetric(metric *pb.Metric) {
	if c.key == "" {
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
		// Добавляем импорт для utils.CalculateHash
		metric.Hash = c.calculateHash([]byte(data))
	}
}

// calculateHash вычисляет хеш используя utils.CalculateHash
func (c *TestMetricsClient) calculateHash(data []byte) string {
	return utils.CalculateHash(data, c.key)
}

func TestGRPCBasicOperations(t *testing.T) {
	// Настройка сервера и клиента
	server, _, listener := setupGRPCServer(t, "", "")
	defer server.Stop()

	client, cleanup := setupGRPCClient(t, listener, "")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Тест ping
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Тест отправки gauge метрики
	gaugeValue := 123.45
	gaugeMetric := models.Metrics{
		ID:    "test_gauge",
		MType: "gauge",
		Value: &gaugeValue,
	}

	if err := client.SendMetric(ctx, gaugeMetric); err != nil {
		t.Fatalf("Failed to send gauge metric: %v", err)
	}

	// Тест получения gauge метрики
	retrievedMetric, err := client.GetMetric(ctx, "test_gauge", "gauge")
	if err != nil {
		t.Fatalf("Failed to get gauge metric: %v", err)
	}

	if retrievedMetric.ID != "test_gauge" || retrievedMetric.MType != "gauge" {
		t.Errorf("Retrieved metric has wrong metadata: %+v", retrievedMetric)
	}

	if retrievedMetric.Value == nil || *retrievedMetric.Value != 123.45 {
		t.Errorf("Retrieved metric has wrong value: expected 123.45, got %v", retrievedMetric.Value)
	}

	// Тест отправки counter метрики
	counterDelta := int64(10)
	counterMetric := models.Metrics{
		ID:    "test_counter",
		MType: "counter",
		Delta: &counterDelta,
	}

	if err := client.SendMetric(ctx, counterMetric); err != nil {
		t.Fatalf("Failed to send counter metric: %v", err)
	}

	// Тест получения counter метрики
	retrievedCounter, err := client.GetMetric(ctx, "test_counter", "counter")
	if err != nil {
		t.Fatalf("Failed to get counter metric: %v", err)
	}

	if retrievedCounter.Delta == nil || *retrievedCounter.Delta != 10 {
		t.Errorf("Retrieved counter has wrong value: expected 10, got %v", retrievedCounter.Delta)
	}
}

func TestGRPCBatchOperations(t *testing.T) {
	// Настройка сервера и клиента
	server, _, listener := setupGRPCServer(t, "", "")
	defer server.Stop()

	client, cleanup := setupGRPCClient(t, listener, "")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Подготовка batch метрик
	gaugeValue1 := 100.0
	gaugeValue2 := 200.0
	counterDelta1 := int64(5)
	counterDelta2 := int64(15)

	metrics := []models.Metrics{
		{ID: "gauge1", MType: "gauge", Value: &gaugeValue1},
		{ID: "gauge2", MType: "gauge", Value: &gaugeValue2},
		{ID: "counter1", MType: "counter", Delta: &counterDelta1},
		{ID: "counter2", MType: "counter", Delta: &counterDelta2},
	}

	// Тест batch отправки
	if err := client.SendMetricsBatch(ctx, metrics); err != nil {
		t.Fatalf("Failed to send metrics batch: %v", err)
	}

	// Проверка что все метрики сохранились
	for _, metric := range metrics {
		retrieved, err := client.GetMetric(ctx, metric.ID, metric.MType)
		if err != nil {
			t.Fatalf("Failed to get metric %s: %v", metric.ID, err)
		}

		switch metric.MType {
		case "gauge":
			if retrieved.Value == nil || *retrieved.Value != *metric.Value {
				t.Errorf("Gauge %s: expected %f, got %v", metric.ID, *metric.Value, retrieved.Value)
			}
		case "counter":
			if retrieved.Delta == nil || *retrieved.Delta != *metric.Delta {
				t.Errorf("Counter %s: expected %d, got %v", metric.ID, *metric.Delta, retrieved.Delta)
			}
		}
	}
}

func TestGRPCTrustedSubnet(t *testing.T) {
	// Настройка сервера с ограниченной подсетью (широкая подсеть для локальной сети)
	server, _, listener := setupGRPCServer(t, "192.168.0.0/16", "")
	defer server.Stop()

	client, cleanup := setupGRPCClient(t, listener, "")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Тест что ping работает (клиент должен быть из localhost)
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping should work for localhost: %v", err)
	}

	// Тест отправки метрики (должна работать)
	gaugeValue := 42.0
	metric := models.Metrics{
		ID:    "trusted_test",
		MType: "gauge",
		Value: &gaugeValue,
	}

	if err := client.SendMetric(ctx, metric); err != nil {
		t.Fatalf("Metric send should work for trusted subnet: %v", err)
	}
}

func TestGRPCWithHashing(t *testing.T) {
	testKey := "test-secret-key"

	// Настройка сервера и клиента с ключом
	server, _, listener := setupGRPCServer(t, "", testKey)
	defer server.Stop()

	client, cleanup := setupGRPCClient(t, listener, testKey)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Тест отправки метрики с корректным хешем
	gaugeValue := 999.99
	metric := models.Metrics{
		ID:    "hashed_gauge",
		MType: "gauge",
		Value: &gaugeValue,
		// В реальном клиенте хеш вычисляется автоматически
	}

	if err := client.SendMetric(ctx, metric); err != nil {
		t.Fatalf("Failed to send metric with hashing: %v", err)
	}

	// Проверяем что метрика сохранилась
	retrieved, err := client.GetMetric(ctx, "hashed_gauge", "gauge")
	if err != nil {
		t.Fatalf("Failed to get hashed metric: %v", err)
	}

	if retrieved.Value == nil || *retrieved.Value != 999.99 {
		t.Errorf("Hashed metric has wrong value: expected 999.99, got %v", retrieved.Value)
	}
}
