package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/gin-gonic/gin"
)

type requestStoreTraceKey struct{}

func TestAPIContexterUsesModulesRequestDataAndCleanup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIContexter())
	cleanupCount := 0
	router.GET("/orgs/:id", func(c *gin.Context) {
		store := reqctx.FromContext(c.Request.Context())
		if store == nil {
			t.Fatal("request scope was not initialized before route action")
		}
		store.AddCleanup(func() { cleanupCount++ })
		api := GetAPIContext(c)
		api.GetData()["form"] = "bound"
		if store.GetData()["form"] != "bound" {
			t.Fatal("Base has a second datastore")
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestStoreTraceKey{}, "current"))
		if api.Value(requestStoreTraceKey{}) != "current" || reqctx.FromContext(c.Request.Context()) != store {
			t.Fatal("middleware lost request context or datastore")
		}
		api.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/orgs/42", nil))
	if response.Code != http.StatusNoContent || cleanupCount != 1 {
		t.Fatalf("status=%d cleanup=%d", response.Code, cleanupCount)
	}
}

type typedIngressKey struct{}

type typedIngressWriter struct {
	http.ResponseWriter
	status int
}

func (w *typedIngressWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func TestTypedAPIContextPreservesHTTPIngressScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := web.NewRouter[*APIContext](engine.Group(""), GetAPIContext)
	cleanupCount := 0
	var first *APIContext
	var wrapped *typedIngressWriter
	router.AfterRouting(
		APIContexter(),
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				wrapped = &typedIngressWriter{ResponseWriter: w}
				next.ServeHTTP(wrapped, req.WithContext(context.WithValue(req.Context(), typedIngressKey{}, "current")))
			})
		},
	)
	router.Get("/orgs/:id", func(api *APIContext) {
		if api.Value(typedIngressKey{}) != "current" ||
			GetAPIContextFromRequest(api.CurrentRequest()) != api {
			t.Error("typed APIContext did not receive the HTTP provider's current request")
		}
		if id, err := api.PositivePathID("id"); err != nil || id != 42 {
			t.Errorf("matched Gin path parameters unavailable: id=%d err=%v", id, err)
		}
		store := reqctx.FromContext(api.CurrentRequest().Context())
		if store == nil {
			t.Fatal("request-local store missing in typed action")
		}
		store.AddCleanup(func() { cleanupCount++ })
		if first == nil {
			first = api
		} else if first == api {
			t.Error("APIContext leaked across requests")
		}
		api.JSON(http.StatusCreated, map[string]bool{"ok": true})
	}, func(*APIContext) {
		t.Error("action executed after committed response")
	})
	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/orgs/42", nil))
		if response.Code != http.StatusCreated || wrapped == nil || wrapped.status != http.StatusCreated {
			t.Fatalf("request %d: status=%d wrapper=%+v", i, response.Code, wrapped)
		}
	}
	if cleanupCount != 2 {
		t.Fatalf("request-local cleanup called %d times, want 2", cleanupCount)
	}
}

func TestTypedHTTPChainPropagatesGinContextMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := web.NewRouter[*APIContext](engine.Group(""), GetAPIContext)
	router.AfterRouting(APIContexter())
	router.Get("/gin-context", append(
		GinActions(func(c *gin.Context) {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestStoreTraceKey{}, "verified"))
		}),
		func(api *APIContext) {
			if api.Value(requestStoreTraceKey{}) != "verified" {
				t.Error("Gin middleware context mutation was lost before the typed action")
			}
			api.Status(http.StatusNoContent)
		},
	)...)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/gin-context", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusNoContent)
	}
}
