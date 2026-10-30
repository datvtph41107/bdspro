package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func TestTrustedProxyClientIP(t *testing.T) {
	tests := []struct {
		name    string
		proxies string
		peer    string
		want    string
	}{
		{"direct connection ignores spoofed forwarded header", "", "10.1.2.3:2345", "10.1.2.3"},
		{"untrusted peer ignores forwarded header", "192.0.2.0/24", "10.1.2.3:2345", "10.1.2.3"},
		{"trusted peer accepts forwarded client address", "10.0.0.0/8", "10.1.2.3:2345", "198.51.100.25"},
		{"comma separated ranges allow precise peer", "192.0.2.1, 10.1.2.3", "10.1.2.3:2345", "198.51.100.25"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
			defer client.Close()
			server, err := NewServer(config.Config{
				Environment:       "development",
				TokenSymmetricKey: strings.Repeat("x", 32),
				TrustedProxies:    tt.proxies,
			}, nil, client)
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}
			server.router.GET("/__test/client_ip", func(c *gin.Context) {
				c.String(http.StatusOK, c.ClientIP())
			})
			request := httptest.NewRequest(http.MethodGet, "/__test/client_ip", nil)
			request.RemoteAddr = tt.peer
			request.Header.Set("X-Forwarded-For", "198.51.100.25")
			response := httptest.NewRecorder()
			server.router.ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Body.String() != tt.want {
				t.Fatalf("got status %d and ip %q, want %q", response.Code, response.Body.String(), tt.want)
			}
		})
	}
}

func TestInvalidTrustedProxyConfigFailsStartup(t *testing.T) {
	for _, proxies := range []string{"not-an-ip", "10.0.0.0/8,,192.0.2.1", "0.0.0.0/0", "::/0"} {
		t.Run(proxies, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
			defer client.Close()
			_, err := NewServer(config.Config{
				Environment:       "development",
				TokenSymmetricKey: strings.Repeat("x", 32),
				TrustedProxies:    proxies,
			}, nil, client)
			if err == nil {
				t.Fatalf("expected invalid TRUSTED_PROXIES %q to stop startup", proxies)
			}
		})
	}
}
