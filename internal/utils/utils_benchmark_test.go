package utils

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"testing"
	"time"
)

// BenchmarkHash_ComputeHash тестирует производительность вычисления хеша
func BenchmarkHash_ComputeHash(b *testing.B) {
	key := "test-key"
	data := "test-data"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := hmac.New(sha256.New, []byte(key))
		h.Write([]byte(data))
		_ = h.Sum(nil)
	}
}

// BenchmarkHash_ComputeHashLarge тестирует производительность вычисления хеша для больших данных
func BenchmarkHash_ComputeHashLarge(b *testing.B) {
	key := "test-key"
	data := make([]byte, 1024*1024) // 1MB данных
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := hmac.New(sha256.New, []byte(key))
		h.Write(data)
		_ = h.Sum(nil)
	}
}

// BenchmarkRetry_Success тестирует производительность retry при успешном выполнении
func BenchmarkRetry_Success(b *testing.B) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Retry(ctx, config, func() error {
			return nil // Успешное выполнение
		})
	}
}

// BenchmarkRetry_Failure тестирует производительность retry при неудачном выполнении
func BenchmarkRetry_Failure(b *testing.B) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Retry(ctx, config, func() error {
			return context.DeadlineExceeded // Всегда неудача
		})
	}
}

// BenchmarkRetry_Timeout тестирует производительность retry с таймаутом
func BenchmarkRetry_Timeout(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	config := DefaultRetryConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Retry(ctx, config, func() error {
			time.Sleep(50 * time.Millisecond) // Имитация долгой операции
			return context.DeadlineExceeded
		})
	}
}

// BenchmarkRetry_ShortTimeout тестирует производительность retry с коротким таймаутом
func BenchmarkRetry_ShortTimeout(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	config := DefaultRetryConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Retry(ctx, config, func() error {
			return context.DeadlineExceeded
		})
	}
}

// BenchmarkRetry_CustomConfig тестирует производительность retry с кастомной конфигурацией
func BenchmarkRetry_CustomConfig(b *testing.B) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Delays:      []time.Duration{time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Retry(ctx, config, func() error {
			return nil
		})
	}
}

// BenchmarkRetry_Parallel тестирует производительность retry в параллельном режиме
func BenchmarkRetry_Parallel(b *testing.B) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Retry(ctx, config, func() error {
				return nil
			})
		}
	})
}

// BenchmarkHash_Parallel тестирует производительность хеширования в параллельном режиме
func BenchmarkHash_Parallel(b *testing.B) {
	key := "test-key"
	data := "test-data"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h := hmac.New(sha256.New, []byte(key))
			h.Write([]byte(data))
			_ = h.Sum(nil)
		}
	})
}
