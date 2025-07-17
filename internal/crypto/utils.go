package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// GenerateKeyPair генерирует пару RSA ключей
func GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("не удалось сгенерировать приватный ключ: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// SavePrivateKeyToFile сохраняет приватный ключ в файл в формате PEM
func SavePrivateKeyToFile(filename string, privateKey *rsa.PrivateKey) error {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("не удалось создать файл приватного ключа: %w", err)
	}
	defer file.Close()

	return pem.Encode(file, privateKeyPEM)
}

// SavePublicKeyToFile сохраняет публичный ключ в файл в формате PEM
func SavePublicKeyToFile(filename string, publicKey *rsa.PublicKey) error {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать публичный ключ: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("не удалось создать файл публичного ключа: %w", err)
	}
	defer file.Close()

	return pem.Encode(file, publicKeyPEM)
}

// EncryptLargeData шифрует данные произвольного размера, разбивая их на блоки
// RSA может шифровать только ограниченное количество данных за раз
func EncryptLargeData(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	// Максимальный размер блока для PKCS1v15 = keySize - 11
	keySize := publicKey.Size()
	blockSize := keySize - 11

	if len(data) <= blockSize {
		// Если данные помещаются в один блок
		return EncryptData(data, publicKey)
	}

	var encrypted []byte
	for i := 0; i < len(data); i += blockSize {
		end := i + blockSize
		if end > len(data) {
			end = len(data)
		}

		block := data[i:end]
		encryptedBlock, err := EncryptData(block, publicKey)
		if err != nil {
			return nil, fmt.Errorf("ошибка шифрования блока: %w", err)
		}

		encrypted = append(encrypted, encryptedBlock...)
	}

	return encrypted, nil
}

// DecryptLargeData дешифрует данные произвольного размера, разбивая их на блоки
func DecryptLargeData(encryptedData []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	keySize := privateKey.Size()

	if len(encryptedData)%keySize != 0 {
		return nil, fmt.Errorf("размер зашифрованных данных должен быть кратен размеру ключа")
	}

	var decrypted []byte
	for i := 0; i < len(encryptedData); i += keySize {
		block := encryptedData[i : i+keySize]
		decryptedBlock, err := DecryptData(block, privateKey)
		if err != nil {
			return nil, fmt.Errorf("ошибка дешифрования блока: %w", err)
		}

		decrypted = append(decrypted, decryptedBlock...)
	}

	return decrypted, nil
}
