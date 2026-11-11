package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// IPActive is transport-independent; security owns the HTTP denial policy.
// An untrusted X-Forwarded-For header must not create a second rate-limit key.
func TestIPActiveHTTPRejectsSpoofedForwardedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is required for IPActive integration")
	}
	action := ipactive.Action("ipactive_http_" + strconv.FormatInt(time.Now().UnixNano(), 10))
	limiter, err := ipactive.NewRedis(client, map[ipactive.Action]ipactive.Policy{
		action: {Mode: ipactive.ModeCount, Limit: 1, Window: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}
	middleware := ipactive.NewMiddleware(limiter)
	const sourceIP = "192.0.2.78"
	hash := sha256.Sum256([]byte(sourceIP))
	key := ipactive.SourceKeyPrefix + string(action) + ":" + hex.EncodeToString(hash[:])
	t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })

	router := web.NewRouter()
	router.AfterRouting(appctx.APIContexter())
	engine := router.GinEngine()
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	engine.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(ipactive.WithMiddleware(c.Request.Context(), middleware))
		c.Next()
	})
	entered := 0
	router.Post("/admit", IPActiveFor(action), func(ctx *appctx.APIContext) {
		entered++
		ctx.Status(http.StatusNoContent)
	})

	for attempt := 0; attempt < 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/admit", nil)
		req.RemoteAddr = sourceIP + ":4567"
		req.Header.Set("X-Forwarded-For", "198.51.100.9")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if attempt == 0 && resp.Code != http.StatusNoContent {
			t.Fatalf("first request denied: %d %s", resp.Code, resp.Body.String())
		}
		if attempt == 1 && (resp.Code != http.StatusTooManyRequests || resp.Header().Get("Retry-After") == "") {
			t.Fatalf("spoofed forwarding header bypassed IPActive: status=%d retry=%q body=%s", resp.Code, resp.Header().Get("Retry-After"), resp.Body.String())
		}
	}
	if entered != 1 {
		t.Fatalf("denied request entered handler: %d calls", entered)
	}
}
