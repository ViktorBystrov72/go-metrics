package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/ViktorBystrov72/go-metrics/internal/config"
	grpcServer "github.com/ViktorBystrov72/go-metrics/internal/grpc"
	"github.com/ViktorBystrov72/go-metrics/internal/logger"
	"github.com/ViktorBystrov72/go-metrics/internal/server"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	pb "github.com/ViktorBystrov72/go-metrics/proto"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

type ServerComponents struct {
	Storage        storage.Storage
	StorageManager *server.StorageManager
	HTTPServer     *http.Server
	PProfServer    *http.Server
	GRPCServer     *grpc.Server
	GRPCListener   net.Listener
}

func printBuildInfo() {
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func initializeStorage(cfg *config.Config) (storage.Storage, error) {
	// Приоритет хранилищ: PostgreSQL -> файл -> память
	if cfg.DatabaseDSN != "" {
		dbStorage, err := storage.NewDatabaseStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		log.Printf("Using PostgreSQL storage")
		return dbStorage, nil
	}

	if cfg.FileStoragePath != "" {
		fileStorage := storage.NewMemStorage()
		if cfg.Restore {
			if err := fileStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
				log.Printf("Failed to load from file: %v", err)
			} else {
				log.Printf("Loaded metrics from file: %s", cfg.FileStoragePath)
			}
		}
		log.Printf("Using file storage: %s", cfg.FileStoragePath)
		return fileStorage, nil
	}

	log.Printf("Using in-memory storage")
	return storage.NewMemStorage(), nil
}

func setupStorageManager(storageInstance storage.Storage, cfg *config.Config) *server.StorageManager {
	storageConfig := &server.Config{
		StoreInterval:   cfg.StoreInterval,
		FileStoragePath: cfg.FileStoragePath,
		Restore:         cfg.Restore,
	}
	storageManager := server.NewStorageManager(storageInstance, storageConfig)
	storageManager.Start()
	return storageManager
}

func setupHTTPServer(cfg *config.Config, storageInstance storage.Storage) (*http.Server, error) {
	router := server.NewRouter(storageInstance, cfg.Key, cfg.CryptoKey, cfg.TrustedSubnet)

	zapLogger, err := logger.NewZapLogger()
	if err != nil {
		return nil, fmt.Errorf("cannot initialize zap logger: %w", err)
	}
	defer zapLogger.Sync()

	loggedRouter := router.WithLogging(zapLogger)

	return &http.Server{
		Addr:    cfg.RunAddr,
		Handler: loggedRouter,
	}, nil
}

func setupPProfServer() *http.Server {
	return &http.Server{
		Addr: "127.0.0.1:6060",
	}
}

func setupGRPCServer(cfg *config.Config, storageInstance storage.Storage) (*grpc.Server, net.Listener, error) {
	// Проверяем, включен ли gRPC сервер
	if !cfg.EnableGRPC {
		return nil, nil, nil
	}

	// Создаем TCP listener для gRPC
	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}

	// Создаем gRPC сервер с middleware
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcServer.LoggingInterceptor(),
			grpcServer.IPCheckInterceptor(cfg.TrustedSubnet),
			grpcServer.RecoveryInterceptor(),
		),
	)

	// Создаем и регистрируем наш MetricsServer
	metricsServer := grpcServer.NewMetricsServer(storageInstance, cfg.Key)
	pb.RegisterMetricsServiceServer(grpcSrv, metricsServer)

	return grpcSrv, listener, nil
}

func startServers(components *ServerComponents) {
	// Запуск pprof на отдельном порту
	go func() {
		if err := components.PProfServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof server error: %v", err)
		}
	}()

	// Запускаем HTTP сервер в горутине
	go func() {
		log.Printf("Запуск HTTP сервера на %s", components.HTTPServer.Addr)
		if err := components.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Запускаем gRPC сервер, если он настроен
	if components.GRPCServer != nil && components.GRPCListener != nil {
		go func() {
			log.Printf("Запуск gRPC сервера на %s", components.GRPCListener.Addr().String())
			if err := components.GRPCServer.Serve(components.GRPCListener); err != nil {
				log.Fatalf("gRPC server error: %v", err)
			}
		}()
	}
}

func waitForShutdownSignal() os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	return <-sigChan
}

func performGracefulShutdown(components *ServerComponents) {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Останавливаем gRPC сервер первым (graceful stop)
	if components.GRPCServer != nil {
		log.Printf("Остановка gRPC сервера...")
		go func() {
			// GracefulStop останавливает прием новых соединений и ждет завершения текущих запросов
			components.GRPCServer.GracefulStop()
		}()

		// Устанавливаем таймаут для graceful stop
		go func() {
			<-shutdownCtx.Done()
			log.Printf("Принудительная остановка gRPC сервера...")
			components.GRPCServer.Stop() // Принудительная остановка если graceful не сработал
		}()
		log.Printf("gRPC сервер остановлен")
	}

	// Останавливаем HTTP сервер
	log.Printf("Остановка HTTP сервера...")
	if err := components.HTTPServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке HTTP сервера: %v", err)
	} else {
		log.Printf("HTTP сервер остановлен")
	}

	// Останавливаем pprof сервер
	log.Printf("Остановка pprof сервера...")
	if err := components.PProfServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке pprof сервера: %v", err)
	} else {
		log.Printf("pprof сервер остановлен")
	}

	// Останавливаем StorageManager
	components.StorageManager.Stop()

	// Принудительно сохраняем данные и закрываем подключения
	if err := components.StorageManager.Shutdown(); err != nil {
		log.Printf("Ошибка при завершении работы с хранилищем: %v", err)
	}

	log.Printf("Сервер успешно завершен")
}

func main() {
	printBuildInfo()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	storageInstance, err := initializeStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Закрываем подключение к базе данных при завершении
	if dbStorage, ok := storageInstance.(interface{ Close() error }); ok {
		defer dbStorage.Close()
	}

	storageManager := setupStorageManager(storageInstance, cfg)

	httpServer, err := setupHTTPServer(cfg, storageInstance)
	if err != nil {
		log.Fatal(err)
	}

	pprofServer := setupPProfServer()

	grpcSrv, grpcListener, err := setupGRPCServer(cfg, storageInstance)
	if err != nil {
		log.Fatal(err)
	}

	components := &ServerComponents{
		Storage:        storageInstance,
		StorageManager: storageManager,
		HTTPServer:     httpServer,
		PProfServer:    pprofServer,
		GRPCServer:     grpcSrv,
		GRPCListener:   grpcListener,
	}

	startServers(components)

	sig := waitForShutdownSignal()
	log.Printf("Получен сигнал %v, запускаем graceful shutdown...", sig)

	performGracefulShutdown(components)
}
