package app

import (
	"flag"
	"fmt"
	"os"
	"strconv"
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
	)

	flag.StringVar(&address, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&reportInterval, "r", 10, "report interval in seconds")
	flag.IntVar(&pollInterval, "p", 2, "poll interval in seconds")
	flag.StringVar(&key, "k", "", "signature key")
	flag.IntVar(&rateLimit, "l", 1, "rate limit for concurrent requests")
	flag.StringVar(&cryptoKey, "crypto-key", "", "path to public key file for encryption")
	flag.Parse()

	if env := os.Getenv("ADDRESS"); env != "" {
		address = env
	}
	if env := os.Getenv("REPORT_INTERVAL"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			reportInterval = v
		} else {
			return nil, fmt.Errorf("invalid REPORT_INTERVAL: %v", env)
		}
	}
	if env := os.Getenv("POLL_INTERVAL"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			pollInterval = v
		} else {
			return nil, fmt.Errorf("invalid POLL_INTERVAL: %v", env)
		}
	}
	if env := os.Getenv("RATE_LIMIT"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			rateLimit = v
		} else {
			return nil, fmt.Errorf("invalid RATE_LIMIT: %v", env)
		}
	}
	if env := os.Getenv("KEY"); env != "" {
		key = env
	}
	if env := os.Getenv("CRYPTO_KEY"); env != "" {
		cryptoKey = env
	}

	if reportInterval <= 0 {
		return nil, fmt.Errorf("REPORT_INTERVAL должен быть больше 0")
	}
	if pollInterval <= 0 {
		return nil, fmt.Errorf("POLL_INTERVAL должен быть больше 0")
	}
	if rateLimit <= 0 {
		return nil, fmt.Errorf("RATE_LIMIT должен быть больше 0")
	}

	return &AgentConfig{
		Address:        address,
		ReportInterval: reportInterval,
		PollInterval:   pollInterval,
		Key:            key,
		RateLimit:      rateLimit,
		CryptoKey:      cryptoKey,
	}, nil
}
