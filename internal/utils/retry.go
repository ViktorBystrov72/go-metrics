package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// RetryConfig содержит настройки для retry логики
type RetryConfig struct {
	MaxAttempts int             // Максимальное количество попыток (включая первую)
	Delays      []time.Duration // Задержки между попытками
}

// DefaultRetryConfig возвращает стандартную конфигурацию retry
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 4,
		Delays:      []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second},
	}
}

// IsRetriableError проверяет, является ли ошибка retriable
func IsRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Проверяем сетевые ошибки
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	// Проверяем PostgreSQL ошибки класса 08 (Connection Exception)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgerrcode.IsConnectionException(pgErr.Code)
	}

	// Проверяем общие сетевые ошибки по тексту
	errText := err.Error()
	switch {
	case errors.Is(err, net.ErrClosed):
		return true
	}

	// Проверяем по тексту ошибки
	retriablePatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no route to host",
		"network is unreachable",
		"timeout",
		"temporary failure",
		"server overloaded",
		"too many connections",
		"connection limit exceeded",
		"temporary error",
	}

	for _, pattern := range retriablePatterns {
		if strings.Contains(strings.ToLower(errText), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// Retry выполняет функцию с retry логикой
func Retry(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		err := fn()
		if err == nil {
			return nil // Успех
		}

		lastErr = err

		if !IsRetriableError(err) {
			return fmt.Errorf("non-retriable error: %w", err)
		}

		if attempt == config.MaxAttempts-1 {
			return fmt.Errorf("max attempts reached (%d), last error: %w", config.MaxAttempts, err)
		}

		delay := config.Delays[attempt]
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry delay: %w", ctx.Err())
		}
	}

	return lastErr
}

// RetryWithBackoff выполняет функцию с экспоненциальным backoff
func RetryWithBackoff(ctx context.Context, maxAttempts int, baseDelay time.Duration, fn func() error) error {
	config := RetryConfig{
		MaxAttempts: maxAttempts,
		Delays:      make([]time.Duration, maxAttempts-1),
	}

	// Создаем экспоненциальный backoff
	for i := 0; i < maxAttempts-1; i++ {
		config.Delays[i] = baseDelay * time.Duration(1<<i) // 1s, 2s, 4s, 8s...
	}

	return Retry(ctx, config, fn)
}
