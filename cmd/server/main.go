package main

import (
	"log"
	"net/http"

	"github.com/ViktorBystrov72/go-metrics/internal/config"
	"github.com/ViktorBystrov72/go-metrics/internal/server"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	var storageInstance storage.Storage

	// Приоритет хранилищ: PostgreSQL -> файл -> память
	if cfg.DatabaseDSN != "" {
		dbStorage, err := storage.NewDatabaseStorage(cfg.DatabaseDSN)
		if err != nil {
			log.Printf("Failed to connect to database: %v, database unavailable", err)
			storageInstance = &storage.BrokenStorage{}
		} else {
			defer dbStorage.Close()
			storageInstance = dbStorage
			log.Printf("Using PostgreSQL storage")
		}
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

	router := server.NewRouter(storageInstance)

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("cannot initialize zap logger: %v", err)
	}
	defer zapLogger.Sync()

	loggedRouter := router.WithLogging(zapLogger)

	log.Fatal(http.ListenAndServe(cfg.RunAddr, loggedRouter))
}
