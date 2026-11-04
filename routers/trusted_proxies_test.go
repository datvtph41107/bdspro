package routers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func TestTrustedIngressPeersDetermineClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		peers  []string
		remote string
		want   string
	}{
		{name: "direct ignores spoofed header", remote: "10.1.2.3:2345", want: "10.1.2.3"},
		{name: "untrusted peer ignores spoofed header", peers: []string{"192.0.2.0/24"}, remote: "10.1.2.3:2345", want: "10.1.2.3"},
		{name: "trusted peer accepts forwarded client", peers: []string{"10.1.2.3"}, remote: "10.1.2.3:2345", want: "198.51.100.25"},
		{name: "trusted subnet accepts forwarded client", peers: []string{"10.1.2.0/24"}, remote: "10.1.2.3:2345", want: "198.51.100.25"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
			defer client.Close()
			server, err := NewServer(setting.Config{
				Environment:       "production",
				TokenSymmetricKey: strings.Repeat("x", 32),
				TrustedProxies:    tc.peers,
			}, nil, client)
			if err != nil {
				t.Fatal(err)
			}
			server.router.GET("/__client_ip_test", func(c *gin.Context) {
				c.String(http.StatusOK, c.ClientIP())
			})
			req := httptest.NewRequest(http.MethodGet, "/__client_ip_test", nil)
			req.RemoteAddr = tc.remote
			req.Header.Set("X-Forwarded-For", "198.51.100.25")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			if w.Code != http.StatusOK || w.Body.String() != tc.want {
				t.Fatalf("remote=%q status=%d client_ip=%q want=%q", tc.remote, w.Code, w.Body.String(), tc.want)
			}
		})
	}
}

func TestUntrustedProxyConfigurationFailsStartup(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	for _, unsafe := range []string{"not-an-ip", "0.0.0.0/0", "::/0", "192.0.2.0/0"} {
		_, err := NewServer(setting.Config{
			TokenSymmetricKey: strings.Repeat("x", 32),
			TrustedProxies:    []string{unsafe},
		}, nil, client)
		if err == nil {
			t.Fatalf("unsafe proxy policy accepted: %q", unsafe)
		}
	}
}
