package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CalculateHash вычисляет HMAC-SHA256 хеш от данных с использованием ключа
func CalculateHash(data []byte, key string) string {
	if key == "" {
		return ""
	}

	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHash проверяет соответствие хеша данным и ключу
func VerifyHash(data []byte, key string, expectedHash string) bool {
	if key == "" || expectedHash == "" {
		return true // если ключ не задан, считаем что проверка прошла
	}

	calculatedHash := CalculateHash(data, key)
	return hmac.Equal([]byte(calculatedHash), []byte(expectedHash))
}
