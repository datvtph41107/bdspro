package web

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
)

// Typed routing must preserve the semantics of the sole production HTTP
// executor (Gin). Parity is tested independently of any Chi-specific phases.
func TestTypedRouterNativeGinParity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		method string
		path   string
		status int
		order  []string
	}{
		{"get", http.MethodGet, "/orgs/42/teams", http.StatusOK, []string{"assignment", "membership", "list"}},
		{"post", http.MethodPost, "/orgs/42/teams", http.StatusCreated, []string{"assignment", "membership", "owner", "create"}},
		{"assignment-aborts", http.MethodGet, "/orgs/42/teams?deny=assignment", http.StatusNotFound, []string{"assignment"}},
		{"membership-aborts", http.MethodPost, "/orgs/42/teams?deny=membership", http.StatusForbidden, []string{"assignment", "membership"}},
		{"owner-aborts", http.MethodPost, "/orgs/42/teams?deny=owner", http.StatusForbidden, []string{"assignment", "membership", "owner"}},
		{"unknown-resource", http.MethodGet, "/orgs/42/unknown", http.StatusNotFound, nil},
		{"wrong-method", http.MethodDelete, "/orgs/42/teams", http.StatusMethodNotAllowed, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			native, nativeOrder := parityRouter(false)
			typed, typedOrder := parityRouter(true)
			nativeResponse := httptest.NewRecorder()
			typedResponse := httptest.NewRecorder()
			native.ServeHTTP(nativeResponse, httptest.NewRequest(tc.method, tc.path, nil))
			typed.ServeHTTP(typedResponse, httptest.NewRequest(tc.method, tc.path, nil))
			if nativeResponse.Code != tc.status || typedResponse.Code != tc.status {
				t.Fatalf("status native=%d typed=%d want=%d",
					nativeResponse.Code, typedResponse.Code, tc.status)
			}
			if nativeResponse.Body.String() != typedResponse.Body.String() {
				t.Fatalf("response drift: native=%q typed=%q",
					nativeResponse.Body.String(), typedResponse.Body.String())
			}
			if !slices.Equal(*nativeOrder, tc.order) || !slices.Equal(*typedOrder, tc.order) {
				t.Fatalf("middleware order native=%v typed=%v want=%v", *nativeOrder, *typedOrder, tc.order)
			}
		})
	}
}

func parityRouter(typed bool) (*gin.Engine, *[]string) {
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	steps := &[]string{}
	guard := func(name string, status int) func(*gin.Context) {
		return func(ctx *gin.Context) {
			*steps = append(*steps, name)
			if ctx.Query("deny") == name {
				ctx.AbortWithStatusJSON(status, gin.H{"code": name + "_denied"})
			}
		}
	}
	assignment := guard("assignment", http.StatusNotFound)
	membership := guard("membership", http.StatusForbidden)
	owner := guard("owner", http.StatusForbidden)
	list := func(ctx *gin.Context) {
		*steps = append(*steps, "list")
		ctx.JSON(http.StatusOK, gin.H{"kind": "list"})
	}
	create := func(ctx *gin.Context) {
		*steps = append(*steps, "create")
		ctx.JSON(http.StatusCreated, gin.H{"kind": "created"})
	}
	if typed {
		router := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
		router.Group("/orgs/:id", func() {
			router.Group("/teams", func() {
				router.Get("", list)
				router.Post("", owner, create)
			}, membership)
		}, assignment)
		return engine, steps
	}
	// A separate, native-Gin route declaration is the behavioral oracle.
	org := engine.Group("/orgs/:id", assignment)
	teams := org.Group("/teams", membership)
	teams.GET("", list)
	teams.POST("", owner, create)
	return engine, steps
}

