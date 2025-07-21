package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPCheckMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		trustedSubnet  string
		realIP         string
		expectedStatus int
		shouldCallNext bool
	}{
		{
			name:           "empty trusted subnet - should pass",
			trustedSubnet:  "",
			realIP:         "192.168.1.10",
			expectedStatus: http.StatusOK,
			shouldCallNext: true,
		},
		{
			name:           "valid IP in subnet - should pass",
			trustedSubnet:  "192.168.1.0/24",
			realIP:         "192.168.1.10",
			expectedStatus: http.StatusOK,
			shouldCallNext: true,
		},
		{
			name:           "invalid IP not in subnet - should reject",
			trustedSubnet:  "192.168.1.0/24",
			realIP:         "10.0.0.1",
			expectedStatus: http.StatusForbidden,
			shouldCallNext: false,
		},
		{
			name:           "missing X-Real-IP header - should reject",
			trustedSubnet:  "192.168.1.0/24",
			realIP:         "",
			expectedStatus: http.StatusForbidden,
			shouldCallNext: false,
		},
		{
			name:           "invalid IP format - should reject",
			trustedSubnet:  "192.168.1.0/24",
			realIP:         "invalid-ip",
			expectedStatus: http.StatusForbidden,
			shouldCallNext: false,
		},
		{
			name:           "localhost in localhost subnet - should pass",
			trustedSubnet:  "127.0.0.0/8",
			realIP:         "127.0.0.1",
			expectedStatus: http.StatusOK,
			shouldCallNext: true,
		},
		{
			name:           "IPv6 address in IPv6 subnet - should pass",
			trustedSubnet:  "2001:db8::/32",
			realIP:         "2001:db8::1",
			expectedStatus: http.StatusOK,
			shouldCallNext: true,
		},
		{
			name:           "IPv6 address not in subnet - should reject",
			trustedSubnet:  "2001:db8::/32",
			realIP:         "2001:db9::1",
			expectedStatus: http.StatusForbidden,
			shouldCallNext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			middleware := IPCheckMiddleware(tt.trustedSubnet)
			handler := middleware(nextHandler)

			req := httptest.NewRequest("POST", "/updates/", nil)
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Ожидался статус %d, получен %d", tt.expectedStatus, w.Code)
			}

			if nextCalled != tt.shouldCallNext {
				t.Errorf("Ожидалось shouldCallNext=%v, получено nextCalled=%v", tt.shouldCallNext, nextCalled)
			}
		})
	}
}

func TestIPCheckMiddleware_InvalidCIDR(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := IPCheckMiddleware("invalid-cidr")
	handler := middleware(nextHandler)

	req := httptest.NewRequest("POST", "/updates/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.10")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Ожидался статус %d для некорректного CIDR, получен %d", http.StatusInternalServerError, w.Code)
	}
}
