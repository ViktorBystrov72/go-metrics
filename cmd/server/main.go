package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/config"
	"github.com/ViktorBystrov72/go-metrics/internal/logger"
	"github.com/ViktorBystrov72/go-metrics/internal/server"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
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
	router := server.NewRouter(storageInstance, cfg.Key, cfg.CryptoKey)

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

func startServers(httpServer, pprofServer *http.Server) {
	// Запуск pprof на отдельном порту
	go func() {
		if err := pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof server error: %v", err)
		}
	}()

	// Запускаем HTTP сервер в горутине
	go func() {
		log.Printf("Запуск сервера на %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
}

func waitForShutdownSignal() os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	return <-sigChan
}

func performGracefulShutdown(components *ServerComponents) {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

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

	components := &ServerComponents{
		Storage:        storageInstance,
		StorageManager: storageManager,
		HTTPServer:     httpServer,
		PProfServer:    pprofServer,
	}

	startServers(httpServer, pprofServer)

	sig := waitForShutdownSignal()
	log.Printf("Получен сигнал %v, запускаем graceful shutdown...", sig)

	performGracefulShutdown(components)
}
