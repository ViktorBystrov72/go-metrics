package app

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/ViktorBystrov72/go-metrics/internal/config"
)

type AgentConfig struct {
	Address        string
	ReportInterval int
	PollInterval   int
	Key            string
	RateLimit      int
	CryptoKey      string
}

func ParseAgentConfig() (*AgentConfig, error) {
	var (
		address        string
		reportInterval int
		pollInterval   int
		key            string
		rateLimit      int
		cryptoKey      string
		configFile     string
	)

	// Создаем отдельный FlagSet чтобы избежать конфликтов с глобальными флагами
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.StringVar(&address, "a", "localhost:8080", "address and port to run server")
	fs.IntVar(&reportInterval, "r", 10, "report interval in seconds")
	fs.IntVar(&pollInterval, "p", 2, "poll interval in seconds")
	fs.StringVar(&key, "k", "", "signature key")
	fs.IntVar(&rateLimit, "l", 1, "rate limit for concurrent requests")
	fs.StringVar(&cryptoKey, "crypto-key", "", "path to public key file for encryption")
	fs.StringVar(&configFile, "c", "", "config file path")
	fs.StringVar(&configFile, "config", "", "config file path") // альтернативный флаг

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

	// Создаем промежуточную структуру для сбора значений из разных источников
	jsonConfig := &config.AgentJSONConfig{}

	// 1. Сначала загружаем JSON конфигурацию (наименьший приоритет)
	if configFile == "" {
		configFile = os.Getenv("CONFIG")
	}

	if configFile != "" {
		err := config.LoadJSONFile(configFile, jsonConfig)
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки JSON конфигурации: %w", err)
		}
	}

	// 2. Применяем переменные окружения (средний приоритет)
	if env := os.Getenv("ADDRESS"); env != "" {
		jsonConfig.Address = stringPtr(env)
	}
	if env := os.Getenv("REPORT_INTERVAL"); env != "" {
		// Если это число без единицы, добавляем "s" (секунды для обратной совместимости)
		if _, err := strconv.Atoi(env); err == nil {
			jsonConfig.ReportInterval = stringPtr(env + "s")
		} else {
			jsonConfig.ReportInterval = stringPtr(env)
		}
	}
	if env := os.Getenv("POLL_INTERVAL"); env != "" {
		// Если это число без единицы, добавляем "s"
		if _, err := strconv.Atoi(env); err == nil {
			jsonConfig.PollInterval = stringPtr(env + "s")
		} else {
			jsonConfig.PollInterval = stringPtr(env)
		}
	}
	if env := os.Getenv("CRYPTO_KEY"); env != "" {
		jsonConfig.CryptoKey = stringPtr(env)
	}

	// KEY и RATE_LIMIT не поддерживаются в JSON, применяем сразу
	if env := os.Getenv("KEY"); env != "" {
		key = env
	}
	if env := os.Getenv("RATE_LIMIT"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			rateLimit = v
		} else {
			return nil, fmt.Errorf("invalid RATE_LIMIT: %v", env)
		}
	}

	// 3. Применяем флаги (наивысший приоритет)
	finalConfig := &config.AgentJSONConfig{}

	// Если флаг был изменен от дефолта, используем его
	if address != "localhost:8080" {
		finalConfig.Address = stringPtr(address)
	}
	if reportInterval != 10 {
		finalConfig.ReportInterval = stringPtr(fmt.Sprintf("%ds", reportInterval))
	}
	if pollInterval != 2 {
		finalConfig.PollInterval = stringPtr(fmt.Sprintf("%ds", pollInterval))
	}
	if cryptoKey != "" {
		finalConfig.CryptoKey = stringPtr(cryptoKey)
	}

	// Применяем JSON конфигурацию для незаданных значений
	jsonConfig.ApplyToAgentConfig(finalConfig)

	// Конвертируем обратно в финальную структуру AgentConfig
	result := &AgentConfig{
		Key:       key,       // KEY не поддерживается в JSON
		RateLimit: rateLimit, // RATE_LIMIT не поддерживается в JSON
	}

	// Обрабатываем значения с дефолтами
	if finalConfig.Address != nil {
		result.Address = *finalConfig.Address
	} else {
		result.Address = "localhost:8080"
	}

	if finalConfig.ReportInterval != nil {
		var err error
		result.ReportInterval, err = config.ParseDurationToSeconds(*finalConfig.ReportInterval)
		if err != nil {
			return nil, fmt.Errorf("некорректный report_interval: %w", err)
		}
	} else {
		result.ReportInterval = 10
	}

	if finalConfig.PollInterval != nil {
		var err error
		result.PollInterval, err = config.ParseDurationToSeconds(*finalConfig.PollInterval)
		if err != nil {
			return nil, fmt.Errorf("некорректный poll_interval: %w", err)
		}
	} else {
		result.PollInterval = 2
	}

	if finalConfig.CryptoKey != nil {
		result.CryptoKey = *finalConfig.CryptoKey
	}

	// Проверяем ограничения
	if result.ReportInterval <= 0 {
		return nil, fmt.Errorf("REPORT_INTERVAL должен быть больше 0")
	}
	if result.PollInterval <= 0 {
		return nil, fmt.Errorf("POLL_INTERVAL должен быть больше 0")
	}
	if result.RateLimit <= 0 {
		return nil, fmt.Errorf("RATE_LIMIT должен быть больше 0")
	}

	return result, nil
}

// stringPtr возвращает указатель на строку
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
