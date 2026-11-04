package ipactive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestGinAdmissionAndSpoofedForwardedIP(t *testing.T) {
	client := redisForTest(t)
	gin.SetMode(gin.TestMode)
	action := Action("ipactive_gin_" + strconv.FormatInt(time.Now().UnixNano(), 10))
	limiter, err := NewRedis(client, map[Action]Policy{
		action: {Mode: ModeCount, Limit: 1, Window: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	calls := 0
	router.POST("/test", NewMiddleware(limiter, func(c *gin.Context, event Event, err error) {
		if err != nil {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		c.Header("Retry-After", strconv.FormatInt(event.RetryAfter, 10))
		c.AbortWithStatus(http.StatusTooManyRequests)
	}).For(action), func(c *gin.Context) {
		calls++
		c.Status(http.StatusNoContent)
	})
	for attempt := 0; attempt < 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "192.0.2.78:4567"
		req.Header.Set("X-Forwarded-For", "198.51.100.9")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if attempt == 0 && w.Code != http.StatusNoContent {
			t.Fatalf("first request: %d", w.Code)
		}
		if attempt == 1 && (w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "") {
			t.Fatalf("second request bypassed IPActive: status=%d retry=%q", w.Code, w.Header().Get("Retry-After"))
		}
	}
	if calls != 1 {
		t.Fatalf("denied request entered handler: %d calls", calls)
	}
	hash := sha256.Sum256([]byte("192.0.2.78"))
	key := SourceKeyPrefix + string(action) + ":" + hex.EncodeToString(hash[:])
	t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })
}
