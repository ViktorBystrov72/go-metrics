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

type flagValues struct {
	address        string
	reportInterval int
	pollInterval   int
	key            string
	rateLimit      int
	cryptoKey      string
	configFile     string
}

func parseFlags() (*flagValues, error) {
	flags := &flagValues{}

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.StringVar(&flags.address, "a", "localhost:8080", "address and port to run server")
	fs.IntVar(&flags.reportInterval, "r", 10, "report interval in seconds")
	fs.IntVar(&flags.pollInterval, "p", 2, "poll interval in seconds")
	fs.StringVar(&flags.key, "k", "", "signature key")
	fs.IntVar(&flags.rateLimit, "l", 1, "rate limit for concurrent requests")
	fs.StringVar(&flags.cryptoKey, "crypto-key", "", "path to public key file for encryption")
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

func loadJSONConfig(configFile string) (*config.AgentJSONConfig, error) {
	jsonConfig := &config.AgentJSONConfig{}

	if configFile == "" {
		configFile = os.Getenv("CONFIG")
	}

	if configFile != "" {
		if err := config.LoadJSONFile(configFile, jsonConfig); err != nil {
			return nil, fmt.Errorf("ошибка загрузки JSON конфигурации: %w", err)
		}
	}

	return jsonConfig, nil
}

func applyEnvironmentVariables(jsonConfig *config.AgentJSONConfig, flags *flagValues) error {
	if env := os.Getenv("ADDRESS"); env != "" {
		jsonConfig.Address = stringPtr(env)
	}

	if env := os.Getenv("REPORT_INTERVAL"); env != "" {
		if _, err := strconv.Atoi(env); err == nil {
			jsonConfig.ReportInterval = stringPtr(env + "s")
		} else {
			jsonConfig.ReportInterval = stringPtr(env)
		}
	}

	if env := os.Getenv("POLL_INTERVAL"); env != "" {
		if _, err := strconv.Atoi(env); err == nil {
			jsonConfig.PollInterval = stringPtr(env + "s")
		} else {
			jsonConfig.PollInterval = stringPtr(env)
		}
	}

	if env := os.Getenv("CRYPTO_KEY"); env != "" {
		jsonConfig.CryptoKey = stringPtr(env)
	}

	// KEY и RATE_LIMIT не поддерживаются в JSON, применяем к флагам
	if env := os.Getenv("KEY"); env != "" {
		flags.key = env
	}

	if env := os.Getenv("RATE_LIMIT"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			flags.rateLimit = v
		} else {
			return fmt.Errorf("invalid RATE_LIMIT: %v", env)
		}
	}

	return nil
}

func applyFlags(flags *flagValues) *config.AgentJSONConfig {
	finalConfig := &config.AgentJSONConfig{}

	// Если флаг был изменен от дефолта, используем его
	if flags.address != "localhost:8080" {
		finalConfig.Address = stringPtr(flags.address)
	}
	if flags.reportInterval != 10 {
		finalConfig.ReportInterval = stringPtr(fmt.Sprintf("%ds", flags.reportInterval))
	}
	if flags.pollInterval != 2 {
		finalConfig.PollInterval = stringPtr(fmt.Sprintf("%ds", flags.pollInterval))
	}
	if flags.cryptoKey != "" {
		finalConfig.CryptoKey = stringPtr(flags.cryptoKey)
	}

	return finalConfig
}

func buildFinalConfig(finalConfig *config.AgentJSONConfig, flags *flagValues) (*AgentConfig, error) {
	result := &AgentConfig{
		Key:       flags.key,
		RateLimit: flags.rateLimit,
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

	return result, nil
}

func validateConfig(cfg *AgentConfig) error {
	if cfg.ReportInterval <= 0 {
		return fmt.Errorf("REPORT_INTERVAL должен быть больше 0")
	}
	if cfg.PollInterval <= 0 {
		return fmt.Errorf("POLL_INTERVAL должен быть больше 0")
	}
	if cfg.RateLimit <= 0 {
		return fmt.Errorf("RATE_LIMIT должен быть больше 0")
	}
	return nil
}

func ParseAgentConfig() (*AgentConfig, error) {
	// 1. Парсим флаги командной строки
	flags, err := parseFlags()
	if err != nil {
		return nil, err
	}

	// 2. Загружаем JSON конфигурацию (наименьший приоритет)
	jsonConfig, err := loadJSONConfig(flags.configFile)
	if err != nil {
		return nil, err
	}

	// 3. Применяем переменные окружения (средний приоритет)
	if err := applyEnvironmentVariables(jsonConfig, flags); err != nil {
		return nil, err
	}

	// 4. Применяем флаги (наивысший приоритет)
	finalConfig := applyFlags(flags)

	// 5. Применяем JSON конфигурацию для незаданных значений
	jsonConfig.ApplyToAgentConfig(finalConfig)

	// 6. Строим финальную конфигурацию
	result, err := buildFinalConfig(finalConfig, flags)
	if err != nil {
		return nil, err
	}

	// 7. Валидируем конфигурацию
	if err := validateConfig(result); err != nil {
		return nil, err
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
