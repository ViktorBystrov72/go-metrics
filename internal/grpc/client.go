package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/converters"
	"github.com/ViktorBystrov72/go-metrics/internal/hashservice"
	"github.com/ViktorBystrov72/go-metrics/internal/models"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// MetricsClient обертка над gRPC клиентом для метрик
type MetricsClient struct {
	client pb.MetricsServiceClient
	conn   *grpc.ClientConn
	hasher *hashservice.ProtoHasher
}

// NewMetricsClient создает новый gRPC клиент для метрик
func NewMetricsClient(serverAddress, key string) (*MetricsClient, error) {
	// Подключаемся к серверу с insecure соединением
	conn, err := grpc.NewClient(serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(ClientIPInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	client := pb.NewMetricsServiceClient(conn)

	return &MetricsClient{
		client: client,
		conn:   conn,
		hasher: hashservice.NewProtoHasher(key),
	}, nil
}

// Close закрывает соединение с gRPC сервером
func (c *MetricsClient) Close() error {
	return c.conn.Close()
}

// SendMetric отправляет одну метрику на сервер
func (c *MetricsClient) SendMetric(ctx context.Context, metric models.Metrics) error {
	// Конвертируем модель в protobuf формат используя общий конвертер
	pbMetric := converters.ModelToProto(metric)

	// Добавляем хеш если ключ задан
	c.hasher.AddHash(pbMetric)

	req := &pb.UpdateMetricRequest{
		Metric: pbMetric,
	}

	resp, err := c.client.UpdateMetric(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send metric: %v", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("server error: %s", resp.Error)
	}

	return nil
}

// SendBatch отправляет множество метрик на сервер в одном запросе
func (c *MetricsClient) SendBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	// Конвертируем все метрики используя общий конвертер
	pbMetrics := converters.ModelsToProtos(metrics)

	// Добавляем хеши ко всем метрикам
	c.hasher.AddHashes(pbMetrics)

	req := &pb.UpdateMetricsRequest{
		Metrics: pbMetrics,
	}

	resp, err := c.client.UpdateMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send metrics batch: %v", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("server error: %s", resp.Error)
	}

	return nil
}

// GetMetric получает значение метрики с сервера
func (c *MetricsClient) GetMetric(ctx context.Context, metricType, metricID string) (*models.Metrics, error) {
	req := &pb.GetMetricRequest{
		Type: metricType,
		Id:   metricID,
	}

	resp, err := c.client.GetMetric(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric: %v", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Metric == nil {
		return nil, fmt.Errorf("no metric returned")
	}

	// Конвертируем protobuf в модель используя общий конвертер
	metric := converters.ProtoToModel(resp.Metric)
	return &metric, nil
}

// GetAllMetrics получает все метрики с сервера
func (c *MetricsClient) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	req := &pb.GetAllMetricsRequest{}

	resp, err := c.client.GetAllMetrics(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get all metrics: %v", err)
	}

	// Конвертируем все protobuf метрики в модели используя общий конвертер
	return converters.ProtosToModels(resp.Metrics), nil
}

// Ping проверяет доступность сервера
func (c *MetricsClient) Ping(ctx context.Context) error {
	req := &pb.PingRequest{}

	resp, err := c.client.Ping(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to ping server: %v", err)
	}

	if !resp.Ok {
		return fmt.Errorf("server not healthy: %s", resp.Error)
	}

	return nil
}

// ClientIPInterceptor добавляет X-Real-IP в метаданные gRPC запроса
func ClientIPInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Получаем локальный IP адрес
		localIP := getLocalIP()
		if localIP != "" {
			// Добавляем IP в метаданные
			md := metadata.Pairs("x-real-ip", localIP)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// getLocalIP получает локальный IP адрес машины
func getLocalIP() string {
	// Пытаемся установить UDP соединение чтобы определить локальный IP
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()

	if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return localAddr.IP.String()
	}

	return ""
}
