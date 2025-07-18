package middleware

import (
	"log"
	"net"
	"net/http"
)

// IPCheckMiddleware создает middleware для проверки IP-адресов против доверенной подсети
func IPCheckMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если доверенная подсеть не указана, пропускаем проверку
			if trustedSubnet == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Получаем IP-адрес из заголовка X-Real-IP
			realIP := r.Header.Get("X-Real-IP")
			if realIP == "" {
				log.Printf("Отсутствует заголовок X-Real-IP")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Парсим IP-адрес
			clientIP := net.ParseIP(realIP)
			if clientIP == nil {
				log.Printf("Некорректный IP-адрес в заголовке X-Real-IP: %s", realIP)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Парсим доверенную подсеть в формате CIDR
			_, subnet, err := net.ParseCIDR(trustedSubnet)
			if err != nil {
				log.Printf("Ошибка парсинга доверенной подсети %s: %v", trustedSubnet, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Проверяем, входит ли IP-адрес клиента в доверенную подсеть
			if !subnet.Contains(clientIP) {
				log.Printf("IP-адрес %s не входит в доверенную подсеть %s", realIP, trustedSubnet)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			log.Printf("IP-адрес %s разрешен (входит в подсеть %s)", realIP, trustedSubnet)
			next.ServeHTTP(w, r)
		})
	}
}
