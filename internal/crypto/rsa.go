package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadPublicKey загружает публичный ключ из PEM файла
func LoadPublicKey(filename string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл публичного ключа: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("не удалось декодировать PEM блок")
	}

	var pub *rsa.PublicKey
	switch block.Type {
	case "PUBLIC KEY":
		pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("не удалось парсить PKIX публичный ключ: %w", err)
		}
		var ok bool
		pub, ok = pubInterface.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("публичный ключ не является RSA ключом")
		}
	case "RSA PUBLIC KEY":
		var err error
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("не удалось парсить PKCS1 публичный ключ: %w", err)
		}
	default:
		return nil, fmt.Errorf("неподдерживаемый тип PEM блока: %s", block.Type)
	}

	return pub, nil
}

// LoadPrivateKey загружает приватный ключ из PEM файла
func LoadPrivateKey(filename string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл приватного ключа: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("не удалось декодировать PEM блок")
	}

	var priv *rsa.PrivateKey
	switch block.Type {
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("не удалось парсить PKCS8 приватный ключ: %w", err)
		}
		var ok bool
		priv, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("приватный ключ не является RSA ключом")
		}
	case "RSA PRIVATE KEY":
		var err error
		priv, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("не удалось парсить PKCS1 приватный ключ: %w", err)
		}
	default:
		return nil, fmt.Errorf("неподдерживаемый тип PEM блока: %s", block.Type)
	}

	return priv, nil
}

// EncryptData шифрует данные с помощью публичного ключа RSA
func EncryptData(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, data)
}

// DecryptData дешифрует данные с помощью приватного ключа RSA
func DecryptData(encryptedData []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedData)
}
