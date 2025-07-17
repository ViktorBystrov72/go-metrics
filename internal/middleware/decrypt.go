package middleware

import (
	"bytes"
	"crypto/rsa"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/ViktorBystrov72/go-metrics/internal/crypto"
)

// DecryptMiddleware создает middleware для дешифрования входящих запросов
func DecryptMiddleware(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если приватный ключ не задан, пропускаем дешифрование
			if privateKey == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Проверяем заголовок, указывающий на зашифрованные данные
			if r.Header.Get("Content-Encoding") != "encrypted" {
				// Если данные не зашифрованы, обрабатываем как обычно
				next.ServeHTTP(w, r)
				return
			}

			encryptedBody, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Декодируем из Base64
			encryptedData, err := base64.StdEncoding.DecodeString(string(encryptedBody))
			if err != nil {
				http.Error(w, "Ошибка декодирования Base64", http.StatusBadRequest)
				return
			}

			// Дешифруем данные
			decryptedData, err := crypto.DecryptLargeData(encryptedData, privateKey)
			if err != nil {
				http.Error(w, "Ошибка дешифрования данных", http.StatusBadRequest)
				return
			}

			// Создаем новое тело запроса с дешифрованными данными
			r.Body = io.NopCloser(bytes.NewReader(decryptedData))
			r.ContentLength = int64(len(decryptedData))

			// Агент шифрует уже сжатые gzip данные, поэтому после дешифрования
			// восстанавливаем заголовок gzip для корректной обработки следующим middleware
			r.Header.Set("Content-Encoding", "gzip")

			next.ServeHTTP(w, r)
		})
	}
}
