package crypto

import (
	"bytes"
	"os"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Ошибка генерации ключевой пары: %v", err)
	}

	if privateKey == nil {
		t.Fatal("Приватный ключ не должен быть nil")
	}

	if publicKey == nil {
		t.Fatal("Публичный ключ не должен быть nil")
	}

	// Проверяем размер ключа
	if privateKey.Size() != 256 { // 2048 bits = 256 bytes
		t.Errorf("Ожидался размер ключа 256 байт, получен %d", privateKey.Size())
	}
}

func TestEncryptDecryptData(t *testing.T) {
	privateKey, publicKey, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Ошибка генерации ключевой пары: %v", err)
	}

	testData := []byte("Hello, RSA encryption!")

	// Шифруем данные
	encrypted, err := EncryptData(testData, publicKey)
	if err != nil {
		t.Fatalf("Ошибка шифрования: %v", err)
	}

	// Проверяем, что зашифрованные данные отличаются от исходных
	if bytes.Equal(testData, encrypted) {
		t.Error("Зашифрованные данные не должны совпадать с исходными")
	}

	// Дешифруем данные
	decrypted, err := DecryptData(encrypted, privateKey)
	if err != nil {
		t.Fatalf("Ошибка дешифрования: %v", err)
	}

	// Проверяем, что дешифрованные данные совпадают с исходными
	if !bytes.Equal(testData, decrypted) {
		t.Errorf("Дешифрованные данные не совпадают с исходными.\nОжидалось: %s\nПолучено: %s", testData, decrypted)
	}
}

func TestEncryptDecryptLargeData(t *testing.T) {
	privateKey, publicKey, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Ошибка генерации ключевой пары: %v", err)
	}

	// Создаем данные больше чем может обработать RSA за один раз
	testData := bytes.Repeat([]byte("Large data for RSA encryption test. "), 20)

	// Шифруем данные
	encrypted, err := EncryptLargeData(testData, publicKey)
	if err != nil {
		t.Fatalf("Ошибка шифрования больших данных: %v", err)
	}

	// Проверяем, что зашифрованные данные отличаются от исходных
	if bytes.Equal(testData, encrypted) {
		t.Error("Зашифрованные данные не должны совпадать с исходными")
	}

	// Дешифруем данные
	decrypted, err := DecryptLargeData(encrypted, privateKey)
	if err != nil {
		t.Fatalf("Ошибка дешифрования больших данных: %v", err)
	}

	// Проверяем, что дешифрованные данные совпадают с исходными
	if !bytes.Equal(testData, decrypted) {
		t.Errorf("Дешифрованные данные не совпадают с исходными.\nОжидалось: %d байт\nПолучено: %d байт", len(testData), len(decrypted))
	}
}

func TestSaveLoadKeys(t *testing.T) {
	privateKeyFile := "test_private.pem"
	publicKeyFile := "test_public.pem"

	// Убираем файлы после теста
	defer func() {
		os.Remove(privateKeyFile)
		os.Remove(publicKeyFile)
	}()

	// Генерируем ключи
	privateKey, publicKey, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Ошибка генерации ключевой пары: %v", err)
	}

	// Сохраняем ключи в файлы
	err = SavePrivateKeyToFile(privateKeyFile, privateKey)
	if err != nil {
		t.Fatalf("Ошибка сохранения приватного ключа: %v", err)
	}

	err = SavePublicKeyToFile(publicKeyFile, publicKey)
	if err != nil {
		t.Fatalf("Ошибка сохранения публичного ключа: %v", err)
	}

	loadedPrivateKey, err := LoadPrivateKey(privateKeyFile)
	if err != nil {
		t.Fatalf("Ошибка загрузки приватного ключа: %v", err)
	}

	loadedPublicKey, err := LoadPublicKey(publicKeyFile)
	if err != nil {
		t.Fatalf("Ошибка загрузки публичного ключа: %v", err)
	}

	// Проверяем, что загруженные ключи работают
	testData := []byte("Test save/load keys")

	encrypted, err := EncryptData(testData, loadedPublicKey)
	if err != nil {
		t.Fatalf("Ошибка шифрования загруженным публичным ключом: %v", err)
	}

	decrypted, err := DecryptData(encrypted, loadedPrivateKey)
	if err != nil {
		t.Fatalf("Ошибка дешифрования загруженным приватным ключом: %v", err)
	}

	if !bytes.Equal(testData, decrypted) {
		t.Errorf("Данные не совпадают после загрузки ключей.\nОжидалось: %s\nПолучено: %s", testData, decrypted)
	}
}

func TestLoadInvalidKeyFile(t *testing.T) {
	// Тестируем загрузку несуществующего файла
	_, err := LoadPrivateKey("nonexistent.pem")
	if err == nil {
		t.Error("Ожидалась ошибка при загрузке несуществующего файла")
	}

	_, err = LoadPublicKey("nonexistent.pem")
	if err == nil {
		t.Error("Ожидалась ошибка при загрузке несуществующего файла")
	}
}
