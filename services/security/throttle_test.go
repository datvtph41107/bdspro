package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

func TestAuthThrottleFailsClosedWithoutRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := web.NewRouter()
	router.AfterRouting(appctx.APIContexter())
	router.Post("/login", func(c *appctx.APIContext) {
		allowed := NewAuthThrottle(nil).AllowAPI(c, "login", "someone@example.test", 30, 10, time.Minute)
		if allowed {
			t.Fatal("auth throttle allowed request without Redis")
		}
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), `"code":"rate_limit_unavailable"`) {
		t.Fatalf("unexpected response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
