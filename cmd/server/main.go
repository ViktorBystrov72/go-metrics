package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

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

	router := server.NewRouter(storageInstance, cfg.Key)

	zapLogger, err := logger.NewZapLogger()
	if err != nil {
		log.Fatalf("cannot initialize zap logger: %v", err)
	}
	defer zapLogger.Sync()

	loggedRouter := router.WithLogging(zapLogger)

	// Запуск pprof на отдельном порту
	go func() {
		if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()

	log.Fatal(http.ListenAndServe(cfg.RunAddr, loggedRouter))
}
