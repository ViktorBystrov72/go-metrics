package ipcheck

import (
	"fmt"
	"log"
	"net"
)

// Result представляет результат проверки IP адреса
type Result struct {
	Allowed bool   // разрешен ли доступ
	Message string // сообщение для логирования
	Error   error  // ошибка если произошла
}

// Service предоставляет функциональность проверки IP адресов
type Service struct {
	trustedSubnet string
	subnet        *net.IPNet // кешированная подсеть для производительности
}

// NewService создает новый IP check service
func NewService(trustedSubnet string) (*Service, error) {
	service := &Service{
		trustedSubnet: trustedSubnet,
	}

	// Если подсеть указана, парсим и кешируем ее
	if trustedSubnet != "" {
		_, subnet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга доверенной подсети %s: %v", trustedSubnet, err)
		}
		service.subnet = subnet
	}

	return service, nil
}

// CheckIP проверяет IP адрес против доверенной подсети
func (s *Service) CheckIP(clientIP string) Result {
	// Если доверенная подсеть не указана, разрешаем доступ
	if s.trustedSubnet == "" {
		return Result{
			Allowed: true,
			Message: "Trusted subnet not configured, allowing access",
		}
	}

	// Проверяем что IP адрес предоставлен
	if clientIP == "" {
		return Result{
			Allowed: false,
			Message: "Отсутствует IP-адрес клиента",
		}
	}

	// Парсим IP-адрес
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return Result{
			Allowed: false,
			Message: fmt.Sprintf("Некорректный IP-адрес: %s", clientIP),
		}
	}

	// Проверяем вхождение IP в подсеть (subnet уже кеширована)
	if !s.subnet.Contains(ip) {
		return Result{
			Allowed: false,
			Message: fmt.Sprintf("IP-адрес %s не входит в доверенную подсеть %s", clientIP, s.trustedSubnet),
		}
	}

	return Result{
		Allowed: true,
		Message: fmt.Sprintf("IP-адрес %s разрешен (входит в подсеть %s)", clientIP, s.trustedSubnet),
	}
}

// LogResult логирует результат проверки IP
func LogResult(result Result) {
	if result.Error != nil {
		log.Printf("IP check error: %v", result.Error)
	} else {
		log.Printf("%s", result.Message)
	}
}
