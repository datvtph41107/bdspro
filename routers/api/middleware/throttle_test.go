package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// Missing Redis must not silently turn authentication throttling off.
func TestSourceLimiterFailsClosedWithoutRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	router := gin.New()
	router.POST("/login", IPActiveSource(nil, "login", 30, time.Minute), func(c *gin.Context) {
		called = true
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	if called {
		t.Fatal("request bypassed the missing Redis limiter")
	}
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), `"code":"rate_limit_unavailable"`) {
		t.Fatalf("unexpected response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAuthThrottleFailsClosedWithoutRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", nil)
	allowed := NewAuthThrottle(nil).Allow(c, "login", "someone@example.test", 30, 10, time.Minute)
	if allowed {
		t.Fatal("auth throttle allowed request without Redis")
	}
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), `"code":"rate_limit_unavailable"`) {
		t.Fatalf("unexpected response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
