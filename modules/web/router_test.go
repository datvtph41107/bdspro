package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTypedGroupOrdering(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	var called []string
	m := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	m.Group("/orgs/:id", func() {
		m.Group("/teams", func() {
			m.Post("", func(*gin.Context) { called = append(called, "owner") }, func(c *gin.Context) {
				called = append(called, "action")
				c.Status(http.StatusCreated)
			})
		}, func(*gin.Context) { called = append(called, "member") })
	}, func(*gin.Context) { called = append(called, "assign") })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orgs/42/teams", nil))
	want := []string{"assign", "member", "owner", "action"}
	if !reflect.DeepEqual(called, want) || w.Code != http.StatusCreated {
		t.Fatalf("order=%v status=%d", called, w.Code)
	}
}

func TestTypedRouterAbortStopsAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	hit := false
	m := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	m.Post("/teams", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": "forbidden"})
	}, func(*gin.Context) { hit = true })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/teams", nil))
	if hit || w.Code != http.StatusForbidden {
		t.Fatalf("action reached=%v status=%d", hit, w.Code)
	}
}

func TestTypedRouterRequiresContextAccessorAtRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	defer func() {
		if recover() == nil {
			t.Fatal("missing accessor must fail during registration, not on a request")
		}
	}()
	_ = NewRouter[*gin.Context](router.Group(""), nil)
}

func TestTypedRouterWrittenResponseStopsFollowingActions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	var called []string
	m := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	m.Get("/written", func(c *gin.Context) {
		called = append(called, "first")
		c.JSON(http.StatusAccepted, gin.H{"accepted": true})
	}, func(*gin.Context) {
		called = append(called, "should-not-run")
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/written", nil))
	if w.Code != http.StatusAccepted || !reflect.DeepEqual(called, []string{"first"}) {
		t.Fatalf("response did not terminate typed route chain: status=%d called=%v", w.Code, called)
	}
}

func TestTypedRouterWrittenGroupGuardStopsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	reached := false
	m := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	m.Group("/teams", func() {
		m.Get("/:team_id", func(*gin.Context) { reached = true })
	}, func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/teams/1", nil))
	if reached || w.Code != http.StatusForbidden {
		t.Fatalf("committed group guard reached endpoint: reached=%v status=%d", reached, w.Code)
	}
}

func TestTypedRouterStatusOnlyDoesNotShortCircuit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	reached := false
	m := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	m.Get("/status", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	}, func(c *gin.Context) {
		reached = true
		c.JSON(http.StatusOK, gin.H{"done": true})
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/status", nil))
	if !reached || w.Code != http.StatusOK {
		t.Fatalf("selected-but-uncommitted status blocked endpoint: reached=%v status=%d", reached, w.Code)
	}
}
