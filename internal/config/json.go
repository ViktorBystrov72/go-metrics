package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// AgentJSONConfig представляет конфигурацию агента в JSON формате
type AgentJSONConfig struct {
	Address        *string `json:"address,omitempty"`
	ReportInterval *string `json:"report_interval,omitempty"`
	PollInterval   *string `json:"poll_interval,omitempty"`
	CryptoKey      *string `json:"crypto_key,omitempty"`
}

// ServerJSONConfig представляет конфигурацию сервера в JSON формате
type ServerJSONConfig struct {
	Address       *string `json:"address,omitempty"`
	Restore       *bool   `json:"restore,omitempty"`
	StoreInterval *string `json:"store_interval,omitempty"`
	StoreFile     *string `json:"store_file,omitempty"`
	DatabaseDSN   *string `json:"database_dsn,omitempty"`
	CryptoKey     *string `json:"crypto_key,omitempty"`
	TrustedSubnet *string `json:"trusted_subnet,omitempty"`
}

// LoadJSONFile загружает и парсит JSON файл конфигурации
func LoadJSONFile(filename string, target interface{}) error {
	if filename == "" {
		return nil // файл не указан
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл конфигурации %s: %w", filename, err)
	}

	err = json.Unmarshal(data, target)
	if err != nil {
		return fmt.Errorf("не удалось парсить JSON конфигурацию из %s: %w", filename, err)
	}

	return nil
}

// ParseDurationToSeconds парсит строку duration и возвращает количество секунд
func ParseDurationToSeconds(durationStr string) (int, error) {
	if durationStr == "" {
		return 0, fmt.Errorf("пустая строка duration")
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return 0, fmt.Errorf("не удалось парсить duration '%s': %w", durationStr, err)
	}

	seconds := int(duration.Seconds())
	if seconds <= 0 {
		return 0, fmt.Errorf("duration должен быть положительным, получен: %s", durationStr)
	}

	return seconds, nil
}

// stringPtr возвращает указатель на строку
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// boolPtr возвращает указатель на bool
func boolPtr(b bool) *bool {
	return &b
}

// ApplyToAgentConfig применяет значения из JSON конфигурации, если они не заданы во flags/env
func (jsonCfg *AgentJSONConfig) ApplyToAgentConfig(cfg *AgentJSONConfig) {
	if cfg.Address == nil && jsonCfg.Address != nil {
		cfg.Address = jsonCfg.Address
	}
	if cfg.ReportInterval == nil && jsonCfg.ReportInterval != nil {
		cfg.ReportInterval = jsonCfg.ReportInterval
	}
	if cfg.PollInterval == nil && jsonCfg.PollInterval != nil {
		cfg.PollInterval = jsonCfg.PollInterval
	}
	if cfg.CryptoKey == nil && jsonCfg.CryptoKey != nil {
		cfg.CryptoKey = jsonCfg.CryptoKey
	}
}

// ApplyToServerConfig применяет значения из JSON конфигурации, если они не заданы во flags/env
func (jsonCfg *ServerJSONConfig) ApplyToServerConfig(cfg *ServerJSONConfig) {
	if cfg.Address == nil && jsonCfg.Address != nil {
		cfg.Address = jsonCfg.Address
	}
	if cfg.Restore == nil && jsonCfg.Restore != nil {
		cfg.Restore = jsonCfg.Restore
	}
	if cfg.StoreInterval == nil && jsonCfg.StoreInterval != nil {
		cfg.StoreInterval = jsonCfg.StoreInterval
	}
	if cfg.StoreFile == nil && jsonCfg.StoreFile != nil {
		cfg.StoreFile = jsonCfg.StoreFile
	}
	if cfg.DatabaseDSN == nil && jsonCfg.DatabaseDSN != nil {
		cfg.DatabaseDSN = jsonCfg.DatabaseDSN
	}
	if cfg.CryptoKey == nil && jsonCfg.CryptoKey != nil {
		cfg.CryptoKey = jsonCfg.CryptoKey
	}
	if cfg.TrustedSubnet == nil && jsonCfg.TrustedSubnet != nil {
		cfg.TrustedSubnet = jsonCfg.TrustedSubnet
	}
}
