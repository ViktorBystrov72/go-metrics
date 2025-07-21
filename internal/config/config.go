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
	TrustedSubnet   string
	GRPCAddr        string // адрес для gRPC сервера
	EnableGRPC      bool   // включить gRPC сервер
}

type serverFlagValues struct {
	runAddr         string
	storeInterval   int
	fileStoragePath string
	restore         bool
	databaseDSN     string
	key             string
	cryptoKey       string
	trustedSubnet   string
	grpcAddr        string
	enableGRPC      bool
	configFile      string
}

func parseServerFlags() (*serverFlagValues, error) {
	flags := &serverFlagValues{}

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.StringVar(&flags.runAddr, "a", "localhost:8080", "address and port to run server")
	fs.IntVar(&flags.storeInterval, "i", 300, "store interval in seconds")
	fs.StringVar(&flags.fileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	fs.BoolVar(&flags.restore, "r", true, "restore from file on start")
	fs.StringVar(&flags.databaseDSN, "d", "", "database DSN")
	fs.StringVar(&flags.key, "k", "", "signature key")
	fs.StringVar(&flags.cryptoKey, "crypto-key", "", "path to private key file for decryption")
	fs.StringVar(&flags.trustedSubnet, "t", "", "trusted subnet in CIDR format")
	fs.StringVar(&flags.grpcAddr, "grpc-addr", "", "address for gRPC server")
	fs.BoolVar(&flags.enableGRPC, "enable-grpc", false, "enable gRPC server")
	fs.StringVar(&flags.configFile, "c", "", "config file path")
	fs.StringVar(&flags.configFile, "config", "", "config file path")

	// Парсим флаги только если это не тестовое окружение
	isTest := len(os.Args) > 0 && (os.Args[0] == "test" ||
		len(os.Args[0]) > 5 && os.Args[0][len(os.Args[0])-5:] == ".test")

	if !isTest && len(os.Args) > 1 {
		if err := fs.Parse(os.Args[1:]); err != nil {
			return nil, fmt.Errorf("ошибка парсинга флагов: %w", err)
		}
	}

	return flags, nil
}

func loadServerJSONConfig(configFile string) (*ServerJSONConfig, error) {
	jsonConfig := &ServerJSONConfig{}

	if configFile == "" {
		configFile = os.Getenv("CONFIG")
	}

	if configFile != "" {
		if err := LoadJSONFile(configFile, jsonConfig); err != nil {
			return nil, fmt.Errorf("ошибка загрузки JSON конфигурации: %w", err)
		}
	}

	return jsonConfig, nil
}

func applyServerEnvironmentVariables(jsonConfig *ServerJSONConfig, flags *serverFlagValues) {
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
		// KEY не поддерживается в JSON, применяем к флагам
		flags.key = envKey
	}

	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		jsonConfig.CryptoKey = stringPtr(envCryptoKey)
	}

	if envTrustedSubnet := os.Getenv("TRUSTED_SUBNET"); envTrustedSubnet != "" {
		jsonConfig.TrustedSubnet = stringPtr(envTrustedSubnet)
	}

	if envGRPCAddr := os.Getenv("GRPC_ADDR"); envGRPCAddr != "" {
		jsonConfig.GRPCAddr = stringPtr(envGRPCAddr)
	}

	if envEnableGRPC := os.Getenv("ENABLE_GRPC"); envEnableGRPC != "" {
		if envEnableGRPC == "true" || envEnableGRPC == "1" {
			jsonConfig.EnableGRPC = boolPtr(true)
		}
	}
}

func applyServerFlags(flags *serverFlagValues) *ServerJSONConfig {
	finalConfig := &ServerJSONConfig{}

	// Если флаг был изменен от дефолта, используем его
	if flags.runAddr != "localhost:8080" {
		finalConfig.Address = stringPtr(flags.runAddr)
	}
	if flags.storeInterval != 300 {
		finalConfig.StoreInterval = stringPtr(fmt.Sprintf("%ds", flags.storeInterval))
	}
	if flags.fileStoragePath != "/tmp/metrics-db.json" {
		finalConfig.StoreFile = stringPtr(flags.fileStoragePath)
	}

	// Обработка restore флага
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if envRestore == "true" || envRestore == "1" {
			finalConfig.Restore = boolPtr(true)
		} else if envRestore == "false" || envRestore == "0" {
			finalConfig.Restore = boolPtr(false)
		}
	}

	if flags.databaseDSN != "" {
		finalConfig.DatabaseDSN = stringPtr(flags.databaseDSN)
	}
	if flags.cryptoKey != "" {
		finalConfig.CryptoKey = stringPtr(flags.cryptoKey)
	}
	if flags.trustedSubnet != "" {
		finalConfig.TrustedSubnet = stringPtr(flags.trustedSubnet)
	}
	if flags.grpcAddr != "" {
		finalConfig.GRPCAddr = stringPtr(flags.grpcAddr)
	}
	if flags.enableGRPC {
		finalConfig.EnableGRPC = boolPtr(true)
	}

	return finalConfig
}

func buildServerConfig(finalConfig *ServerJSONConfig, flags *serverFlagValues) (*Config, error) {
	result := &Config{
		Key: flags.key, // KEY не поддерживается в JSON
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

	if finalConfig.TrustedSubnet != nil {
		result.TrustedSubnet = *finalConfig.TrustedSubnet
	}

	if finalConfig.GRPCAddr != nil {
		result.GRPCAddr = *finalConfig.GRPCAddr
	} else {
		// Значение по умолчанию для gRPC адреса если включен gRPC
		if finalConfig.EnableGRPC != nil && *finalConfig.EnableGRPC {
			result.GRPCAddr = "localhost:9090"
		}
	}

	if finalConfig.EnableGRPC != nil {
		result.EnableGRPC = *finalConfig.EnableGRPC
	}

	return result, nil
}

func validateServerConfig(cfg *Config) error {
	if cfg.StoreInterval < 0 {
		return fmt.Errorf("STORE_INTERVAL must be non-negative, got %d", cfg.StoreInterval)
	}
	return nil
}

// Load загружает конфигурацию из флагов, переменных окружения и JSON файла
func Load() (*Config, error) {
	// 1. Парсим флаги командной строки
	flags, err := parseServerFlags()
	if err != nil {
		return nil, err
	}

	// 2. Загружаем JSON конфигурацию (наименьший приоритет)
	jsonConfig, err := loadServerJSONConfig(flags.configFile)
	if err != nil {
		return nil, err
	}

	// 3. Применяем переменные окружения (средний приоритет)
	applyServerEnvironmentVariables(jsonConfig, flags)

	// 4. Применяем флаги (наивысший приоритет)
	finalConfig := applyServerFlags(flags)

	// 5. Применяем JSON конфигурацию для незаданных значений
	jsonConfig.ApplyToServerConfig(finalConfig)

	// 6. Строим финальную конфигурацию
	result, err := buildServerConfig(finalConfig, flags)
	if err != nil {
		return nil, err
	}

	// 7. Валидируем конфигурацию
	if err := validateServerConfig(result); err != nil {
		return nil, err
	}

	return result, nil
}
