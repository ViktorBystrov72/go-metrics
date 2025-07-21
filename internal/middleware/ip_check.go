package middleware

import (
	"log"
	"net/http"

	"github.com/ViktorBystrov72/go-metrics/internal/ipcheck"
)

// IPCheckMiddleware создает middleware для проверки IP-адресов против доверенной подсети
func IPCheckMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	// Создаем IP check service
	ipService, err := ipcheck.NewService(trustedSubnet)
	if err != nil {
		log.Printf("Ошибка создания IP check service: %v", err)
		// Возвращаем middleware который всегда запрещает доступ
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем IP-адрес из заголовка X-Real-IP
			realIP := r.Header.Get("X-Real-IP")

			// Проверяем IP через общий сервис
			result := ipService.CheckIP(realIP)

			// Логируем результат
			ipcheck.LogResult(result)

			// Проверяем результат
			if !result.Allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
