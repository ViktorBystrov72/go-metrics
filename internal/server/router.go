package server

import (
	"crypto/rsa"
	"log"
	"net/http"

	"github.com/ViktorBystrov72/go-metrics/internal/crypto"
	"github.com/ViktorBystrov72/go-metrics/internal/logger"
	"github.com/ViktorBystrov72/go-metrics/internal/middleware"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Router настраивает HTTP роутер
type Router struct {
	handlers *Handlers
	router   *chi.Mux
}

// NewRouter создает новый роутер
func NewRouter(storage storage.Storage, key string, cryptoKeyPath string) *Router {
	handlers := NewHandlers(storage, key)
	router := chi.NewRouter()

	var privateKey *rsa.PrivateKey

	// Загружаем приватный ключ, если путь указан
	if cryptoKeyPath != "" {
		var err error
		privateKey, err = crypto.LoadPrivateKey(cryptoKeyPath)
		if err != nil {
			log.Printf("Ошибка загрузки приватного ключа: %v", err)
			// Продолжаем работу без дешифрования
		} else {
			log.Printf("Приватный ключ загружен из: %s", cryptoKeyPath)
		}
	}

	// Middleware
	router.Use(middleware.DecryptMiddleware(privateKey))
	router.Use(middleware.GzipMiddleware)

	// Маршруты для обновления метрик
	router.Route("/update", func(r chi.Router) {
		r.Post("/{type}/{name}/{value}", handlers.UpdateHandler)
	})

	// Маршруты для получения значений метрик
	router.Route("/value", func(r chi.Router) {
		r.Get("/{type}/{name}", handlers.ValueHandler)
	})

	// Главная страница со списком всех метрик
	router.Get("/", handlers.IndexHandler)

	// Проверка соединения с базой данных
	router.Get("/ping", handlers.PingHandler)

	// JSON API
	router.Post("/update/", handlers.UpdateJSONHandler)
	router.Post("/value/", handlers.ValueJSONHandler)
	router.Post("/updates/", handlers.UpdatesHandler)

	return &Router{
		handlers: handlers,
		router:   router,
	}
}

// WithLogging добавляет логирование к роутеру
func (r *Router) WithLogging(zapLogger *zap.Logger) http.Handler {
	return logger.WithLogging(zapLogger, r.router)
}

// GetRouter возвращает настроенный роутер
func (r *Router) GetRouter() *chi.Mux {
	return r.router
}
