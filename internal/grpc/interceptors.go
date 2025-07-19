package grpc

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// IPCheckInterceptor создает gRPC interceptor для проверки IP-адресов против доверенной подсети
func IPCheckInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Если доверенная подсеть не указана, пропускаем проверку
		if trustedSubnet == "" {
			return handler(ctx, req)
		}

		// Получаем IP-адрес из метаданных X-Real-IP или из peer info
		var clientIP string

		// Сначала пытаемся получить из метаданных (заголовок X-Real-IP)
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ips := md.Get("x-real-ip"); len(ips) > 0 {
				clientIP = ips[0]
			}
		}

		// Если не найден в метаданных, получаем из peer info
		if clientIP == "" {
			if p, ok := peer.FromContext(ctx); ok {
				if addr, ok := p.Addr.(*net.TCPAddr); ok {
					clientIP = addr.IP.String()
				}
			}
		}

		if clientIP == "" {
			log.Printf("Отсутствует X-Real-IP в метаданных и peer info")
			return nil, status.Error(codes.PermissionDenied, "Forbidden")
		}

		// Парсим IP-адрес
		ip := net.ParseIP(clientIP)
		if ip == nil {
			log.Printf("Некорректный IP-адрес: %s", clientIP)
			return nil, status.Error(codes.PermissionDenied, "Forbidden")
		}

		// Парсим доверенную подсеть
		_, subnet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			log.Printf("Ошибка парсинга доверенной подсети %s: %v", trustedSubnet, err)
			return nil, status.Error(codes.Internal, "Internal server error")
		}

		// Проверяем вхождение IP в подсеть
		if !subnet.Contains(ip) {
			log.Printf("IP-адрес %s не входит в доверенную подсеть %s", clientIP, trustedSubnet)
			return nil, status.Error(codes.PermissionDenied, "Forbidden")
		}

		log.Printf("IP-адрес %s разрешен (входит в подсеть %s)", clientIP, trustedSubnet)
		return handler(ctx, req)
	}
}

// LoggingInterceptor создает gRPC interceptor для логирования запросов
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Printf("gRPC request: %s", info.FullMethod)

		resp, err := handler(ctx, req)

		if err != nil {
			log.Printf("gRPC error for %s: %v", info.FullMethod, err)
		} else {
			log.Printf("gRPC success for %s", info.FullMethod)
		}

		return resp, err
	}
}

// AuthInterceptor создает gRPC interceptor для аутентификации (пример)
func AuthInterceptor(secretKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Если ключ не задан, пропускаем аутентификацию
		if secretKey == "" {
			return handler(ctx, req)
		}

		// Получаем токен из метаданных
		var token string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if tokens := md.Get("authorization"); len(tokens) > 0 {
				token = tokens[0]
			}
		}

		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		if token != "Bearer "+secretKey {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization token")
		}

		return handler(ctx, req)
	}
}
