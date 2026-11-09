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
    if !reflect.DeepEqual(called, want) || w.Code != http.StatusCreated { t.Fatalf("order=%v status=%d", called, w.Code) }
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
    if hit || w.Code != http.StatusForbidden { t.Fatalf("action reached=%v status=%d", hit, w.Code) }
}
