package ipactive

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestDefaultPolicyParityAndValidation(t *testing.T) {
	policies := DefaultPolicies()
	if len(policies) != 21 {
		t.Fatalf("expected 21 existing route actions, got %d", len(policies))
	}
	for action, policy := range policies {
		if !validAction(action) || policy.Validate() != nil || policy.Mode != ModeCount {
			t.Fatalf("invalid or changed default %s: %+v", action, policy)
		}
	}
	if policies[Register] != (Policy{Mode: ModeCount, Limit: 10, Window: time.Hour}) {
		t.Fatal("register policy changed")
	}
	if policies[Login] != (Policy{Mode: ModeCount, Limit: 30, Window: time.Minute}) {
		t.Fatal("login policy changed")
	}
	policies[Register] = Policy{}
	if DefaultPolicies()[Register].Limit != 10 {
		t.Fatal("default policy map was shared")
	}
}

func TestInvalidPolicies(t *testing.T) {
	for _, policy := range []Policy{
		{Mode: ModeCount, Limit: 0, Window: time.Minute},
		{Mode: ModeCount, Limit: 1, Window: 0},
		{Mode: ModeCooldown, Limit: 2, Window: time.Minute},
		{Mode: "other", Limit: 1, Window: time.Minute},
	} {
		if policy.Validate() == nil {
			t.Fatalf("unsafe policy accepted: %+v", policy)
		}
	}
	if validAction("login:admin") || validAction("") || !validAction(Login) {
		t.Fatal("unsafe action accepted")
	}
	if _, err := NewRedis(nil, DefaultPolicies()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected fail-closed nil Redis: %v", err)
	}
}

func TestGinMiddlewareFailsClosedWithoutLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	router := gin.New()
	router.POST("/login", NewMiddleware(nil, func(c *gin.Context, ev Event, err error) {
		if ev.Action != Login || !errors.Is(err, ErrUnavailable) {
			t.Errorf("incorrect unavailable event: %+v %v", ev, err)
		}
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": "rate_limit_unavailable"})
	}).For(Login), func(c *gin.Context) {
		called = true
		c.Status(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	if called || recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("admission bypass: downstream=%v status=%d", called, recorder.Code)
	}
}

func TestGinMiddlewareRequiresErrorWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	router := gin.New()
	router.POST("/login", NewMiddleware(nil, nil).For(Login), func(c *gin.Context) {
		called = true
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	if called || recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("misconfigured middleware bypass: downstream=%v status=%d", called, recorder.Code)
	}
}
