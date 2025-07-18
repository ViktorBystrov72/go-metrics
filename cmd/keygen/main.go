package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ViktorBystrov72/go-metrics/internal/crypto"
)

func main() {
	var (
		privateKeyFile = flag.String("private", "private.pem", "путь к файлу приватного ключа")
		publicKeyFile  = flag.String("public", "public.pem", "путь к файлу публичного ключа")
		keySize        = flag.Int("size", 2048, "размер ключа в битах")
	)
	flag.Parse()

	fmt.Printf("Генерация RSA ключей размером %d бит...\n", *keySize)

	// Генерируем ключи
	privateKey, publicKey, err := crypto.GenerateKeyPair(*keySize)
	if err != nil {
		log.Fatalf("Ошибка генерации ключей: %v", err)
	}

	// Сохраняем приватный ключ
	err = crypto.SavePrivateKeyToFile(*privateKeyFile, privateKey)
	if err != nil {
		log.Fatalf("Ошибка сохранения приватного ключа: %v", err)
	}
	fmt.Printf("Приватный ключ сохранен в: %s\n", *privateKeyFile)

	// Сохраняем публичный ключ
	err = crypto.SavePublicKeyToFile(*publicKeyFile, publicKey)
	if err != nil {
		log.Fatalf("Ошибка сохранения публичного ключа: %v", err)
	}
	fmt.Printf("Публичный ключ сохранен в: %s\n", *publicKeyFile)

	fmt.Println("Генерация ключей завершена успешно!")
}
