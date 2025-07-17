package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config содержит конфигурацию сервера
type Config struct {
	RunAddr         string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
	CryptoKey       string
}

// Load загружает конфигурацию из флагов и переменных окружения
func Load() (*Config, error) {
	var (
		flagRunAddr         string
		flagStoreInterval   int
		flagFileStoragePath string
		flagRestore         bool
		flagDatabaseDSN     string
		flagKey             string
		flagCryptoKey       string
	)

	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&flagStoreInterval, "i", 300, "store interval in seconds")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&flagRestore, "r", true, "restore from file on start")
	flag.StringVar(&flagDatabaseDSN, "d", "", "database DSN")
	flag.StringVar(&flagKey, "k", "", "signature key")
	flag.StringVar(&flagCryptoKey, "crypto-key", "", "path to private key file for decryption")
	flag.Parse()

	// Приоритет: env > flag > default
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		if v, err := strconv.Atoi(envStoreInterval); err == nil {
			flagStoreInterval = v
		}
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		flagFileStoragePath = envFileStoragePath
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if envRestore == "true" || envRestore == "1" {
			flagRestore = true
		} else if envRestore == "false" || envRestore == "0" {
			flagRestore = false
		}
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		flagDatabaseDSN = envDatabaseDSN
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		flagKey = envKey
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		flagCryptoKey = envCryptoKey
	}

	if flagStoreInterval < 0 {
		return nil, fmt.Errorf("STORE_INTERVAL must be non-negative, got %d", flagStoreInterval)
	}

	return &Config{
		RunAddr:         flagRunAddr,
		StoreInterval:   flagStoreInterval,
		FileStoragePath: flagFileStoragePath,
		Restore:         flagRestore,
		DatabaseDSN:     flagDatabaseDSN,
		Key:             flagKey,
		CryptoKey:       flagCryptoKey,
	}, nil
}
