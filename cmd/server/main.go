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

	storage := storage.NewMemStorage()

	storageConfig := &server.Config{
		StoreInterval:   cfg.StoreInterval,
		FileStoragePath: cfg.FileStoragePath,
		Restore:         cfg.Restore,
	}
	storageManager := server.NewStorageManager(storage, storageConfig)
	storageManager.Start()

	router := server.NewRouter(storage)

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("cannot initialize zap logger: %v", err)
	}
	defer zapLogger.Sync()

	loggedRouter := router.WithLogging(zapLogger)

	log.Fatal(http.ListenAndServe(cfg.RunAddr, loggedRouter))
}
