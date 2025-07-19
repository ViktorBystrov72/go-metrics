package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/utils"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// MetricsClient обертка над gRPC клиентом для метрик
type MetricsClient struct {
	client pb.MetricsServiceClient
	conn   *grpc.ClientConn
	key    string // ключ для подписи метрик
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
		key:    key,
	}, nil
}

// Close закрывает соединение с сервером
func (c *MetricsClient) Close() error {
	return c.conn.Close()
}

// SendMetric отправляет одну метрику на сервер
func (c *MetricsClient) SendMetric(ctx context.Context, metric models.Metrics) error {
	pbMetric := c.modelToProto(metric)
	c.addHashToMetric(pbMetric)

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

// SendMetricsBatch отправляет множество метрик на сервер одним запросом
func (c *MetricsClient) SendMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	pbMetrics := make([]*pb.Metric, 0, len(metrics))
	for _, metric := range metrics {
		pbMetric := c.modelToProto(metric)
		c.addHashToMetric(pbMetric)
		pbMetrics = append(pbMetrics, pbMetric)
	}

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
func (c *MetricsClient) GetMetric(ctx context.Context, id, metricType string) (*models.Metrics, error) {
	req := &pb.GetMetricRequest{
		Id:   id,
		Type: metricType,
	}

	resp, err := c.client.GetMetric(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric: %v", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Metric == nil {
		return nil, fmt.Errorf("metric not found")
	}

	return c.protoToModel(resp.Metric), nil
}

// GetAllMetrics получает все метрики с сервера
func (c *MetricsClient) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	req := &pb.GetAllMetricsRequest{}

	resp, err := c.client.GetAllMetrics(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get all metrics: %v", err)
	}

	metrics := make([]models.Metrics, 0, len(resp.Metrics))
	for _, pbMetric := range resp.Metrics {
		metrics = append(metrics, *c.protoToModel(pbMetric))
	}

	return metrics, nil
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

// modelToProto конвертирует внутреннюю модель в protobuf формат
func (c *MetricsClient) modelToProto(metric models.Metrics) *pb.Metric {
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

// protoToModel конвертирует protobuf формат во внутреннюю модель
func (c *MetricsClient) protoToModel(pbMetric *pb.Metric) *models.Metrics {
	metric := &models.Metrics{
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

// addHashToMetric добавляет хеш к метрике если ключ задан
func (c *MetricsClient) addHashToMetric(metric *pb.Metric) {
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
		metric.Hash = utils.CalculateHash([]byte(data), c.key)
	}
}

// ClientIPInterceptor добавляет X-Real-IP к исходящим запросам
func ClientIPInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Получаем IP хоста
		hostIP := getHostIP()

		// Добавляем IP в метаданные
		md := metadata.New(map[string]string{
			"x-real-ip": hostIP,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Вызываем оригинальный метод с обогащенным контекстом
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// getHostIP получает IP-адрес хоста для заголовка X-Real-IP
func getHostIP() string {
	// Пытаемся подключиться к внешнему адресу для определения локального IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// Если не удалось, используем localhost
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// ClientWithTimeout создает контекст с таймаутом для gRPC запросов
func ClientWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
