package utils

import (
	"fmt"
)

// ExampleCalculateHash демонстрирует вычисление хеша от данных.
func ExampleCalculateHash() {
	data := []byte("test data")
	key := "secret-key"

	hash := CalculateHash(data, key)
	fmt.Printf("Hash length: %d\n", len(hash))
	fmt.Printf("Hash is not empty: %t\n", hash != "")

	// Output:
	// Hash length: 64
	// Hash is not empty: true
}

// ExampleVerifyHash демонстрирует проверку хеша.
func ExampleVerifyHash() {
	data := []byte("test data")
	key := "secret-key"

	// Вычисляем хеш
	hash := CalculateHash(data, key)

	// Проверяем хеш
	isValid := VerifyHash(data, key, hash)
	fmt.Printf("Hash verification: %t\n", isValid)

	// Output:
	// Hash verification: true
}

// ExampleCalculateHash_emptyKey демонстрирует вычисление хеша с пустым ключом.
func ExampleCalculateHash_emptyKey() {
	data := []byte("test data")
	key := ""

	hash := CalculateHash(data, key)
	fmt.Printf("Hash with empty key: %s\n", hash)

	// Output:
	// Hash with empty key:
}

// ExampleVerifyHash_emptyKey демонстрирует проверку хеша с пустым ключом.
func ExampleVerifyHash_emptyKey() {
	data := []byte("test data")
	key := ""
	expectedHash := ""

	isValid := VerifyHash(data, key, expectedHash)
	fmt.Printf("Verification with empty key: %t\n", isValid)

	// Output:
	// Verification with empty key: true
}

// ExampleVerifyHash_invalidHash демонстрирует проверку неверного хеша.
func ExampleVerifyHash_invalidHash() {
	data := []byte("test data")
	key := "secret-key"
	invalidHash := "invalid-hash"

	isValid := VerifyHash(data, key, invalidHash)
	fmt.Printf("Verification with invalid hash: %t\n", isValid)

	// Output:
	// Verification with invalid hash: false
}
