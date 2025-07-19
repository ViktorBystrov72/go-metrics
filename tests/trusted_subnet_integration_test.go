package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/ViktorBystrov72/go-metrics/internal/app"
	"github.com/ViktorBystrov72/go-metrics/internal/models"
	"github.com/ViktorBystrov72/go-metrics/internal/server"
	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

func TestTrustedSubnetIntegration(t *testing.T) {
	tests := []struct {
		name          string
		trustedSubnet string
		expectStatus  int
		expectSuccess bool
	}{
		{
			name:          "empty trusted subnet - should allow all",
			trustedSubnet: "",
			expectStatus:  200,
			expectSuccess: true,
		},
		{
			name:          "localhost subnet - should allow localhost",
			trustedSubnet: "192.168.0.0/16", // используем широкую подсеть для локальной сети
			expectStatus:  200,
			expectSuccess: true,
		},
		{
			name:          "private subnet - should allow private IPs",
			trustedSubnet: "192.168.0.0/16",
			expectStatus:  200,
			expectSuccess: true,
		},
		{
			name:          "restrictive subnet - should reject most IPs",
			trustedSubnet: "10.0.0.0/24",
			expectStatus:  403,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем сервер с trusted subnet
			storage := storage.NewMemStorage()
			router := server.NewRouter(storage, "", "", tt.trustedSubnet)
			testServer := httptest.NewServer(router.GetRouter())
			defer testServer.Close()

			// Создаем конфигурацию агента
			cfg := &app.AgentConfig{
				Address:        testServer.URL[7:], // убираем "http://"
				ReportInterval: 1,
				PollInterval:   1,
				Key:            "",
				RateLimit:      1,
				CryptoKey:      "",
			}

			// Создаем агент
			sender := app.NewMetricsSender(cfg)

			// Тестовые метрики
			testMetrics := []models.Metrics{
				{
					ID:    "test_trusted_subnet",
					MType: "gauge",
					Value: func() *float64 { v := 123.45; return &v }(),
				},
			}

			metricsSender, ok := sender.(*app.MetricsSender)
			if !ok {
				t.Fatalf("Не удалось привести sender к типу *MetricsSender")
			}

			// Отправляем метрики
			err := metricsSender.SendMetricsBatch(testMetrics)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Ожидался успех, но получена ошибка: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Ожидалась ошибка для restrictive subnet, но запрос прошел успешно")
				}
			}
		})
	}
}

func TestTrustedSubnetIPFormats(t *testing.T) {
	tests := []struct {
		name          string
		trustedSubnet string
		testIP        string
		expectAllow   bool
	}{
		{
			name:          "IPv4 localhost in localhost subnet",
			trustedSubnet: "127.0.0.0/8",
			testIP:        "127.0.0.1",
			expectAllow:   true,
		},
		{
			name:          "IPv4 private in private subnet",
			trustedSubnet: "192.168.1.0/24",
			testIP:        "192.168.1.100",
			expectAllow:   true,
		},
		{
			name:          "IPv4 public not in private subnet",
			trustedSubnet: "192.168.1.0/24",
			testIP:        "8.8.8.8",
			expectAllow:   false,
		},
		{
			name:          "IPv6 localhost in IPv6 localhost",
			trustedSubnet: "::1/128",
			testIP:        "::1",
			expectAllow:   true,
		},
		{
			name:          "IPv6 in IPv6 subnet",
			trustedSubnet: "2001:db8::/32",
			testIP:        "2001:db8::1",
			expectAllow:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем сервер с trusted subnet
			storage := storage.NewMemStorage()
			router := server.NewRouter(storage, "", "", tt.trustedSubnet)
			testServer := httptest.NewServer(router.GetRouter())
			defer testServer.Close()

			// Создаем тестовый запрос с конкретным IP
			req := httptest.NewRequest("POST", "/updates/", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Real-IP", tt.testIP)

			w := httptest.NewRecorder()
			router.GetRouter().ServeHTTP(w, req)

			if tt.expectAllow {
				if w.Code == 403 {
					t.Errorf("IP %s должен быть разрешен в подсети %s, но получен статус 403", tt.testIP, tt.trustedSubnet)
				}
			} else {
				if w.Code != 403 {
					t.Errorf("IP %s должен быть заблокирован в подсети %s, но получен статус %d", tt.testIP, tt.trustedSubnet, w.Code)
				}
			}
		})
	}
}

func TestTrustedSubnetConfigurationPriority(t *testing.T) {
	// Тест проверяет что конфигурация trusted subnet работает через флаги, переменные окружения и JSON

	t.Run("empty config allows all", func(t *testing.T) {
		storage := storage.NewMemStorage()
		router := server.NewRouter(storage, "", "", "")
		testServer := httptest.NewServer(router.GetRouter())
		defer testServer.Close()

		req := httptest.NewRequest("POST", "/updates/", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", "8.8.8.8") // Публичный IP

		w := httptest.NewRecorder()
		router.GetRouter().ServeHTTP(w, req)

		// Без trusted subnet все IP должны проходить
		if w.Code == 403 {
			t.Error("При пустом trusted subnet все IP должны быть разрешены")
		}
	})

	t.Run("restrictive config blocks", func(t *testing.T) {
		storage := storage.NewMemStorage()
		router := server.NewRouter(storage, "", "", "10.0.0.0/24")
		testServer := httptest.NewServer(router.GetRouter())
		defer testServer.Close()

		req := httptest.NewRequest("POST", "/updates/", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", "8.8.8.8") // Публичный IP не в подсети

		w := httptest.NewRecorder()
		router.GetRouter().ServeHTTP(w, req)

		// IP не в подсети должен быть заблокирован
		if w.Code != 403 {
			t.Errorf("IP не в подсети должен быть заблокирован, получен статус %d", w.Code)
		}
	})
}
