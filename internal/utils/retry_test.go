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
			name:     "retriable network error",
			err:      &MockRetriableError{message: "connection refused"},
			expected: true,
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
	fn := func() error {
		return errors.New("permanent error")
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

	if !errors.Is(err, errors.New("max attempts reached (3), last error: permanent error")) {
		t.Errorf("Unexpected error: %v", err)
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
