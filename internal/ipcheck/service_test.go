package ipcheck

import (
	"testing"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name          string
		trustedSubnet string
		expectError   bool
	}{
		{
			name:          "empty subnet",
			trustedSubnet: "",
			expectError:   false,
		},
		{
			name:          "valid CIDR",
			trustedSubnet: "192.168.0.0/16",
			expectError:   false,
		},
		{
			name:          "invalid CIDR",
			trustedSubnet: "invalid-cidr",
			expectError:   true,
		},
		{
			name:          "single IP as CIDR",
			trustedSubnet: "127.0.0.1/32",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.trustedSubnet)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && service == nil {
				t.Errorf("Expected service but got nil")
			}
		})
	}
}

func TestService_CheckIP(t *testing.T) {
	tests := []struct {
		name          string
		trustedSubnet string
		clientIP      string
		expectAllowed bool
	}{
		// Случай без доверенной подсети
		{
			name:          "no trusted subnet",
			trustedSubnet: "",
			clientIP:      "1.2.3.4",
			expectAllowed: true,
		},

		// Валидные IP в подсети
		{
			name:          "IP in subnet",
			trustedSubnet: "192.168.0.0/16",
			clientIP:      "192.168.1.100",
			expectAllowed: true,
		},
		{
			name:          "IP exactly on network",
			trustedSubnet: "10.0.0.0/8",
			clientIP:      "10.0.0.1",
			expectAllowed: true,
		},
		{
			name:          "localhost in loopback",
			trustedSubnet: "127.0.0.0/8",
			clientIP:      "127.0.0.1",
			expectAllowed: true,
		},

		// IP не в подсети
		{
			name:          "IP not in subnet",
			trustedSubnet: "192.168.0.0/16",
			clientIP:      "10.0.0.1",
			expectAllowed: false,
		},
		{
			name:          "public IP with private subnet",
			trustedSubnet: "192.168.0.0/24",
			clientIP:      "8.8.8.8",
			expectAllowed: false,
		},

		// Невалидные IP
		{
			name:          "empty IP",
			trustedSubnet: "192.168.0.0/16",
			clientIP:      "",
			expectAllowed: false,
		},
		{
			name:          "invalid IP",
			trustedSubnet: "192.168.0.0/16",
			clientIP:      "invalid-ip",
			expectAllowed: false,
		},
		{
			name:          "malformed IP",
			trustedSubnet: "192.168.0.0/16",
			clientIP:      "999.999.999.999",
			expectAllowed: false,
		},

		// IPv6 тесты
		{
			name:          "IPv6 in subnet",
			trustedSubnet: "2001:db8::/32",
			clientIP:      "2001:db8::1",
			expectAllowed: true,
		},
		{
			name:          "IPv6 not in subnet",
			trustedSubnet: "2001:db8::/32",
			clientIP:      "2001:db9::1",
			expectAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.trustedSubnet)
			if err != nil {
				t.Fatalf("Failed to create service: %v", err)
			}

			result := service.CheckIP(tt.clientIP)

			if result.Allowed != tt.expectAllowed {
				t.Errorf("Expected allowed=%v, got allowed=%v. Message: %s",
					tt.expectAllowed, result.Allowed, result.Message)
			}

			// Проверяем что сообщение не пустое
			if result.Message == "" {
				t.Errorf("Expected non-empty message")
			}
		})
	}
}

func TestLogResult(t *testing.T) {
	// Простой тест что LogResult не паникует
	results := []Result{
		{Allowed: true, Message: "Test allowed"},
		{Allowed: false, Message: "Test denied"},
		{Allowed: false, Message: "Test error", Error: nil},
	}

	for _, result := range results {
		// Должно работать без паники
		LogResult(result)
	}
}
