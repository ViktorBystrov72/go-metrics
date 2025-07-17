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

// Load загружает конфигурацию из флагов, переменных окружения и JSON файла
func Load() (*Config, error) {
	var (
		flagRunAddr         string
		flagStoreInterval   int
		flagFileStoragePath string
		flagRestore         bool
		flagDatabaseDSN     string
		flagKey             string
		flagCryptoKey       string
		flagConfigFile      string
	)

	// Создаем отдельный FlagSet чтобы избежать конфликтов с глобальными флагами
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	fs.IntVar(&flagStoreInterval, "i", 300, "store interval in seconds")
	fs.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	fs.BoolVar(&flagRestore, "r", true, "restore from file on start")
	fs.StringVar(&flagDatabaseDSN, "d", "", "database DSN")
	fs.StringVar(&flagKey, "k", "", "signature key")
	fs.StringVar(&flagCryptoKey, "crypto-key", "", "path to private key file for decryption")
	fs.StringVar(&flagConfigFile, "c", "", "config file path")
	fs.StringVar(&flagConfigFile, "config", "", "config file path") // альтернативный флаг

	// Парсим флаги только если это не тестовое окружение
	// В тестах os.Args[0] обычно заканчивается на ".test"
	isTest := len(os.Args) > 0 && (os.Args[0] == "test" ||
		len(os.Args[0]) > 5 && os.Args[0][len(os.Args[0])-5:] == ".test")

	if !isTest && len(os.Args) > 1 {
		err := fs.Parse(os.Args[1:])
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга флагов: %w", err)
		}
	}

	jsonConfig := &ServerJSONConfig{}

	// 1. Сначала загружаем JSON конфигурацию (наименьший приоритет)
	configFile := flagConfigFile
	if configFile == "" {
		configFile = os.Getenv("CONFIG")
	}

	if configFile != "" {
		err := LoadJSONFile(configFile, jsonConfig)
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки JSON конфигурации: %w", err)
		}
	}

	// 2. Применяем переменные окружения (средний приоритет)
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		jsonConfig.Address = stringPtr(envRunAddr)
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		// Если это число без единицы, добавляем "s" (секунды для обратной совместимости)
		if _, err := strconv.Atoi(envStoreInterval); err == nil {
			jsonConfig.StoreInterval = stringPtr(envStoreInterval + "s")
		} else {
			jsonConfig.StoreInterval = stringPtr(envStoreInterval)
		}
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		jsonConfig.StoreFile = stringPtr(envFileStoragePath)
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if envRestore == "true" || envRestore == "1" {
			jsonConfig.Restore = boolPtr(true)
		} else if envRestore == "false" || envRestore == "0" {
			jsonConfig.Restore = boolPtr(false)
		}
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		jsonConfig.DatabaseDSN = stringPtr(envDatabaseDSN)
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		// KEY не поддерживается в JSON, применяем сразу
		flagKey = envKey
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		jsonConfig.CryptoKey = stringPtr(envCryptoKey)
	}

	// 3. Применяем флаги (наивысший приоритет)
	finalConfig := &ServerJSONConfig{}

	// Если флаг был изменен от дефолта, используем его
	if flagRunAddr != "localhost:8080" {
		finalConfig.Address = stringPtr(flagRunAddr)
	}
	if flagStoreInterval != 300 {
		finalConfig.StoreInterval = stringPtr(fmt.Sprintf("%ds", flagStoreInterval))
	}
	if flagFileStoragePath != "/tmp/metrics-db.json" {
		finalConfig.StoreFile = stringPtr(flagFileStoragePath)
	}

	// Для restore нужно проверить, был ли флаг явно установлен

	// Если есть env переменная, используем её приоритет
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		// Env переменная установлена, используем её значение
		if envRestore == "true" || envRestore == "1" {
			finalConfig.Restore = boolPtr(true)
		} else if envRestore == "false" || envRestore == "0" {
			finalConfig.Restore = boolPtr(false)
		}
	}
	// Если нет env переменной, оставляем finalConfig.Restore = nil
	// JSON конфигурация сможет примениться в ApplyToServerConfig

	if flagDatabaseDSN != "" {
		finalConfig.DatabaseDSN = stringPtr(flagDatabaseDSN)
	}
	if flagCryptoKey != "" {
		finalConfig.CryptoKey = stringPtr(flagCryptoKey)
	}

	// Применяем JSON конфигурацию для незаданных значений
	jsonConfig.ApplyToServerConfig(finalConfig)

	// Конвертируем обратно в финальную структуру Config
	result := &Config{
		Key: flagKey, // KEY не поддерживается в JSON
	}

	// Обрабатываем значения с дефолтами
	if finalConfig.Address != nil {
		result.RunAddr = *finalConfig.Address
	} else {
		result.RunAddr = "localhost:8080"
	}

	if finalConfig.StoreInterval != nil {
		var err error
		result.StoreInterval, err = ParseDurationToSeconds(*finalConfig.StoreInterval)
		if err != nil {
			return nil, fmt.Errorf("некорректный store_interval: %w", err)
		}
	} else {
		result.StoreInterval = 300
	}

	if finalConfig.StoreFile != nil {
		result.FileStoragePath = *finalConfig.StoreFile
	} else {
		result.FileStoragePath = "/tmp/metrics-db.json"
	}

	if finalConfig.Restore != nil {
		result.Restore = *finalConfig.Restore
	} else {
		result.Restore = true
	}

	if finalConfig.DatabaseDSN != nil {
		result.DatabaseDSN = *finalConfig.DatabaseDSN
	}

	if finalConfig.CryptoKey != nil {
		result.CryptoKey = *finalConfig.CryptoKey
	}

	// Проверяем ограничения
	if result.StoreInterval < 0 {
		return nil, fmt.Errorf("STORE_INTERVAL must be non-negative, got %d", result.StoreInterval)
	}

	return result, nil
}
