package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockError - мок ошибка для тестирования
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// MockRetriableError - мок retriable ошибка
type MockRetriableError struct {
	message string
}

func (e *MockRetriableError) Error() string {
	return e.message
}

func (e *MockRetriableError) Temporary() bool {
	return true
}

func (e *MockRetriableError) Timeout() bool {
	return false
}

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-retriable error",
			err:      errors.New("permanent error"),
			expected: false,
		},
		{
			name:     "connection refused text",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection reset text",
			err:      errors.New("connection reset"),
			expected: true,
		},
		{
			name:     "broken pipe text",
			err:      errors.New("broken pipe"),
			expected: true,
		},
		{
			name:     "timeout text",
			err:      errors.New("timeout"),
			expected: true,
		},
		{
			name:     "server overloaded text",
			err:      errors.New("server overloaded"),
			expected: true,
		},
		{
			name:     "too many connections text",
			err:      errors.New("too many connections"),
			expected: true,
		},
		{
			name:     "temporary error text",
			err:      errors.New("temporary error"),
			expected: true,
		},
		{
			name:     "case insensitive connection refused",
			err:      errors.New("CONNECTION REFUSED"),
			expected: true,
		},
		{
			name:     "case insensitive timeout",
			err:      errors.New("TIMEOUT ERROR"),
			expected: true,
		},
		{
			name:     "mixed case server overloaded",
			err:      errors.New("Server Overloaded"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetriableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetriableError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRetry_Success(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	config := RetryConfig{
		MaxAttempts: 4,
		Delays:      []time.Duration{1 * time.Millisecond, 1 * time.Millisecond, 1 * time.Millisecond},
	}

	ctx := context.Background()
	err := Retry(ctx, config, fn)

	if err != nil {
		t.Errorf("Retry() returned error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_MaxAttemptsReached(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("temporary error")
	}

	config := RetryConfig{
		MaxAttempts: 3,
		Delays:      []time.Duration{1 * time.Millisecond, 1 * time.Millisecond},
	}

	ctx := context.Background()
	err := Retry(ctx, config, fn)

	if err == nil {
		t.Error("Retry() should return error when max attempts reached")
	}

	expectedErrorText := "max attempts reached (3), last error: temporary error"
	if err.Error() != expectedErrorText {
		t.Errorf("Expected error: %s, got: %s", expectedErrorText, err.Error())
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_NonRetriableError(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("permanent error")
	}

	config := RetryConfig{
		MaxAttempts: 4,
		Delays:      []time.Duration{1 * time.Millisecond, 1 * time.Millisecond, 1 * time.Millisecond},
	}

	ctx := context.Background()
	err := Retry(ctx, config, fn)

	if err == nil {
		t.Error("Retry() should return error for non-retriable error")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retriable error, got %d", attempts)
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	fn := func() error {
		return errors.New("temporary error")
	}

	config := RetryConfig{
		MaxAttempts: 4,
		Delays:      []time.Duration{100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем контекст сразу

	err := Retry(ctx, config, fn)

	if err == nil {
		t.Error("Retry() should return error when context is cancelled")
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 4 {
		t.Errorf("Expected MaxAttempts = 4, got %d", config.MaxAttempts)
	}

	if len(config.Delays) != 3 {
		t.Errorf("Expected 3 delays, got %d", len(config.Delays))
	}

	expectedDelays := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	for i, delay := range config.Delays {
		if delay != expectedDelays[i] {
			t.Errorf("Expected delay[%d] = %v, got %v", i, expectedDelays[i], delay)
		}
	}
}

func TestRetryWithBackoff(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	ctx := context.Background()
	err := RetryWithBackoff(ctx, 4, 10*time.Millisecond, fn)

	if err != nil {
		t.Errorf("RetryWithBackoff() returned error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryWithContextCancellation тестирует retry с отменой контекста.
func TestRetryWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attempts := 0
	err := Retry(ctx, DefaultRetryConfig(), func() error {
		attempts++
		if attempts == 2 {
			cancel() // Отменяем контекст
		}
		return errors.New("test error")
	})

	if err == nil {
		t.Error("Ожидалась ошибка при отмене контекста")
	}

	if attempts < 1 {
		t.Errorf("Ожидалась хотя бы 1 попытка, выполнено %d", attempts)
	}
}

// TestRetryWithImmediateSuccess тестирует retry с немедленным успехом.
func TestRetryWithImmediateSuccess(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := Retry(ctx, DefaultRetryConfig(), func() error {
		attempts++
		return nil // Успех с первой попытки
	})

	if err != nil {
		t.Errorf("Не ожидалась ошибка: %v", err)
	}

	if attempts != 1 {
		t.Errorf("Ожидалась 1 попытка, выполнено %d", attempts)
	}
}

// TestRetryWithBackoffSuccess тестирует retry с backoff и успехом.
func TestRetryWithBackoffSuccess(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := RetryWithBackoff(ctx, 3, 100*time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil // Успех на третьей попытке
	})

	if err != nil {
		t.Errorf("Не ожидалась ошибка: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Ожидалось 3 попытки, выполнено %d", attempts)
	}
}

// TestRetryWithBackoffMaxAttempts тестирует retry с backoff и максимальным количеством попыток.
func TestRetryWithBackoffMaxAttempts(t *testing.T) {
	ctx := context.Background()

	attempts := 0
	err := RetryWithBackoff(ctx, 2, 100*time.Millisecond, func() error {
		attempts++
		return errors.New("persistent error")
	})

	if err == nil {
		t.Error("Ожидалась ошибка после максимального количества попыток")
	}

	if attempts < 1 {
		t.Errorf("Ожидалась хотя бы 1 попытка, выполнено %d", attempts)
	}
}

// TestRetryWithBackoffContextCancellation тестирует retry с backoff и отменой контекста.
func TestRetryWithBackoffContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attempts := 0
	err := RetryWithBackoff(ctx, 5, 100*time.Millisecond, func() error {
		attempts++
		if attempts == 2 {
			cancel() // Отменяем контекст
		}
		return errors.New("test error")
	})

	if err == nil {
		t.Error("Ожидалась ошибка при отмене контекста")
	}

	if attempts < 1 {
		t.Errorf("Ожидалась хотя бы 1 попытка, выполнено %d", attempts)
	}
}

// TestIsRetriableErrorWithNil тестирует IsRetriableError с nil ошибкой.
func TestIsRetriableErrorWithNil(t *testing.T) {
	if IsRetriableError(nil) {
		t.Error("nil ошибка не должна быть retriable")
	}
}

// TestIsRetriableErrorWithNonRetriable тестирует IsRetriableError с не-retriable ошибкой.
func TestIsRetriableErrorWithNonRetriable(t *testing.T) {
	err := errors.New("permanent error")
	if IsRetriableError(err) {
		t.Error("Постоянная ошибка не должна быть retriable")
	}
}

// TestRetryConfigValidation тестирует валидацию конфигурации retry.
func TestRetryConfigValidation(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts <= 0 {
		t.Error("MaxAttempts должен быть больше 0")
	}

	if len(config.Delays) == 0 {
		t.Error("Delays не должен быть пустым")
	}
}

// TestRetryWithZeroMaxAttempts тестирует retry с нулевым MaxAttempts.
func TestRetryWithZeroMaxAttempts(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.MaxAttempts = 0

	attempts := 0
	err := Retry(ctx, config, func() error {
		attempts++
		return errors.New("test error")
	})

	// При нулевом MaxAttempts функция должна вернуть nil (не выполнять попытки)
	if err != nil {
		t.Errorf("При нулевом MaxAttempts функция должна вернуть nil, получено: %v", err)
	}

	// Проверяем, что попыток не было
	if attempts != 0 {
		t.Errorf("При нулевом MaxAttempts не должно быть попыток, выполнено %d", attempts)
	}
}

// TestRetryWithBackoffZeroMaxAttempts тестирует retry с backoff и нулевым MaxAttempts.
func TestRetryWithBackoffZeroMaxAttempts(t *testing.T) {
	ctx := context.Background()

	attempts := 0
	defer func() {
		if r := recover(); r == nil {
			t.Error("Ожидалась паника при нулевом MaxAttempts")
		}
	}()
	_ = RetryWithBackoff(ctx, 0, 100*time.Millisecond, func() error {
		attempts++
		return errors.New("test error")
	})
}
