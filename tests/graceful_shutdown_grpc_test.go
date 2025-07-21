package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcServer "github.com/ViktorBystrov72/go-metrics/internal/grpc"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
)

// TestGRPCGracefulShutdown проверяет корректное завершение работы gRPC сервера
func TestGRPCGracefulShutdown(t *testing.T) {
	// Создаем in-memory storage
	store := storage.NewMemStorage()

	// Создаем gRPC сервер
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcServer.LoggingInterceptor(),
			grpcServer.RecoveryInterceptor(),
		),
	)

	// Регистрируем сервис
	metricsServer := grpcServer.NewMetricsServer(store, "")
	pb.RegisterMetricsServiceServer(grpcSrv, metricsServer)

	// Создаем listener
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Запускаем сервер в горутине
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		if err := grpcSrv.Serve(listener); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Создаем клиент
	conn, err := grpc.NewClient(listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewMetricsServiceClient(conn)

	// Проверяем что сервер работает
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Тестируем graceful shutdown
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		grpcSrv.GracefulStop()
	}()

	// Проверяем что shutdown завершился за разумное время
	select {
	case <-shutdownDone:
		t.Log("Graceful shutdown completed successfully")
	case <-time.After(5 * time.Second):
		t.Error("Graceful shutdown took too long")
		grpcSrv.Stop() // Принудительная остановка
	}

	// Проверяем что сервер действительно остановился
	select {
	case <-serverDone:
		t.Log("Server goroutine finished")
	case <-time.After(2 * time.Second):
		t.Error("Server goroutine did not finish")
	}
}

// TestGRPCShutdownWithActiveConnections проверяет shutdown с активными соединениями
func TestGRPCShutdownWithActiveConnections(t *testing.T) {
	store := storage.NewMemStorage()

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcServer.LoggingInterceptor(),
		),
	)

	metricsServer := grpcServer.NewMetricsServer(store, "")
	pb.RegisterMetricsServiceServer(grpcSrv, metricsServer)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Запускаем сервер
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		grpcSrv.Serve(listener)
	}()

	// Создаем клиент
	conn, err := grpc.NewClient(listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewMetricsServiceClient(conn)

	// Запускаем долгий запрос в горутине
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err := client.Ping(ctx, &pb.PingRequest{})
		if err != nil {
			t.Logf("Request failed during shutdown: %v", err)
		}
	}()

	// Даем запросу время начаться
	time.Sleep(100 * time.Millisecond)

	// Инициируем graceful shutdown
	start := time.Now()
	grpcSrv.GracefulStop()
	shutdownTime := time.Since(start)

	// Graceful shutdown должен корректно завершить активные запросы
	t.Logf("Shutdown completed in %v", shutdownTime)

	select {
	case <-requestDone:
		t.Log("Active request completed")
	case <-time.After(5 * time.Second):
		t.Error("Active request did not complete")
	}

	select {
	case <-serverDone:
		t.Log("Server shutdown completed")
	case <-time.After(2 * time.Second):
		t.Error("Server did not shutdown")
	}
}
