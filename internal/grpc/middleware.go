package grpc

import (
	"context"
	"log"

	"github.com/ViktorBystrov72/go-metrics/internal/crypto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// CryptoInterceptor создает gRPC interceptor для дешифрования данных (если включено)
func CryptoInterceptor(privateKeyPath string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Если путь к приватному ключу не указан, пропускаем дешифрование
		if privateKeyPath == "" {
			return handler(ctx, req)
		}

		// Загружаем приватный ключ для проверки валидности пути
		_, err := crypto.LoadPrivateKey(privateKeyPath)
		if err != nil {
			log.Printf("Ошибка загрузки приватного ключа: %v", err)
			// Продолжаем работу без дешифрования
			return handler(ctx, req)
		}

		// Получаем метаданные для проверки шифрования
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			// Нет метаданных, обрабатываем как обычно
			return handler(ctx, req)
		}

		// Проверяем заголовок Content-Encoding
		encodings := md.Get("content-encoding")
		if len(encodings) == 0 || encodings[0] != "encrypted" {
			// Данные не зашифрованы, обрабатываем как обычно
			return handler(ctx, req)
		}

		// В gRPC данные уже десериализованы из protobuf,
		// так что дешифрование должно происходить на уровне транспорта
		// Пока что просто логируем и продолжаем
		log.Printf("Получены зашифрованные данные для метода %s", info.FullMethod)

		return handler(ctx, req)
	}
}

// ChainUnaryInterceptors объединяет несколько interceptors в один
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	if len(interceptors) == 0 {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	if len(interceptors) == 1 {
		return interceptors[0]
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Создаем цепочку вызовов
		chainHandler := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			currentHandler := chainHandler
			chainHandler = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
					return currentHandler(ctx, req)
				})
			}
		}

		return chainHandler(ctx, req)
	}
}

// SetupServerInterceptors настраивает все interceptors для gRPC сервера
func SetupServerInterceptors(trustedSubnet, cryptoKeyPath string) grpc.UnaryServerInterceptor {
	var interceptors []grpc.UnaryServerInterceptor

	// Добавляем логирование (первым, чтобы логировать все запросы)
	interceptors = append(interceptors, LoggingInterceptor())

	// Добавляем проверку доверенных IP адресов
	if trustedSubnet != "" {
		interceptors = append(interceptors, IPCheckInterceptor(trustedSubnet))
	}

	// Добавляем дешифрование
	if cryptoKeyPath != "" {
		interceptors = append(interceptors, CryptoInterceptor(cryptoKeyPath))
	}

	return ChainUnaryInterceptors(interceptors...)
}

// RecoveryInterceptor обеспечивает graceful recovery от паник в gRPC handlers
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered in gRPC handler %s: %v", info.FullMethod, r)
				err = status.Error(codes.Internal, "internal server error")
				resp = nil
			}
		}()

		return handler(ctx, req)
	}
}
