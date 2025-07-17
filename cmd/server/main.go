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

func main() {
	printBuildInfo()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	var storageInstance storage.Storage

	// Приоритет хранилищ: PostgreSQL -> файл -> память
	if cfg.DatabaseDSN != "" {
		dbStorage, err := storage.NewDatabaseStorage(cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer dbStorage.Close()
		storageInstance = dbStorage
		log.Printf("Using PostgreSQL storage")
	} else if cfg.FileStoragePath != "" {
		fileStorage := storage.NewMemStorage()
		if cfg.Restore {
			if err := fileStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
				log.Printf("Failed to load from file: %v", err)
			} else {
				log.Printf("Loaded metrics from file: %s", cfg.FileStoragePath)
			}
		}
		storageInstance = fileStorage
		log.Printf("Using file storage: %s", cfg.FileStoragePath)
	} else {
		storageInstance = storage.NewMemStorage()
		log.Printf("Using in-memory storage")
	}

	storageConfig := &server.Config{
		StoreInterval:   cfg.StoreInterval,
		FileStoragePath: cfg.FileStoragePath,
		Restore:         cfg.Restore,
	}
	storageManager := server.NewStorageManager(storageInstance, storageConfig)
	storageManager.Start()

	router := server.NewRouter(storageInstance, cfg.Key, cfg.CryptoKey)

	zapLogger, err := logger.NewZapLogger()
	if err != nil {
		log.Fatalf("cannot initialize zap logger: %v", err)
	}
	defer zapLogger.Sync()

	loggedRouter := router.WithLogging(zapLogger)

	// Создаем HTTP сервер
	srv := &http.Server{
		Addr:    cfg.RunAddr,
		Handler: loggedRouter,
	}

	// Запуск pprof на отдельном порту
	pprofServer := &http.Server{
		Addr: "127.0.0.1:6060",
	}
	go func() {
		if err := pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof server error: %v", err)
		}
	}()

	// Канал для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Запуск сервера на %s", cfg.RunAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Ожидаем сигнал остановки
	sig := <-sigChan
	log.Printf("Получен сигнал %v, запускаем graceful shutdown...", sig)

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Останавливаем HTTP сервер
	log.Printf("Остановка HTTP сервера...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке HTTP сервера: %v", err)
	} else {
		log.Printf("HTTP сервер остановлен")
	}

	// Останавливаем pprof сервер
	log.Printf("Остановка pprof сервера...")
	if err := pprofServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке pprof сервера: %v", err)
	} else {
		log.Printf("pprof сервер остановлен")
	}

	// Останавливаем StorageManager
	storageManager.Stop()

	// Принудительно сохраняем данные и закрываем подключения
	if err := storageManager.Shutdown(); err != nil {
		log.Printf("Ошибка при завершении работы с хранилищем: %v", err)
	}

	log.Printf("Сервер успешно завершен")
}
