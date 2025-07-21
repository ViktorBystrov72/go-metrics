package grpc

import (
	"context"
	"log"
	"net"

	"github.com/ViktorBystrov72/go-metrics/internal/ipcheck"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// IPCheckInterceptor создает gRPC interceptor для проверки IP-адресов против доверенной подсети
func IPCheckInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	// Создаем IP check service
	ipService, err := ipcheck.NewService(trustedSubnet)
	if err != nil {
		log.Printf("Ошибка создания IP check service: %v", err)
		// Возвращаем interceptor который всегда запрещает доступ
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return nil, status.Error(codes.Internal, "Internal server error")
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
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

		// Проверяем IP через общий сервис
		result := ipService.CheckIP(clientIP)

		// Логируем результат
		ipcheck.LogResult(result)

		// Проверяем результат
		if !result.Allowed {
			return nil, status.Error(codes.PermissionDenied, "Forbidden")
		}

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
