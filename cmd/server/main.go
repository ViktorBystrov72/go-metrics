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

	// Если указан DSN для базы данных, используем PostgreSQL
	if cfg.DatabaseDSN != "" {
		dbStorage, err := storage.NewDatabaseStorage(cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer dbStorage.Close()
		storageInstance = dbStorage
	} else {
		// Иначе используем хранилище в памяти
		storageInstance = storage.NewMemStorage()
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
