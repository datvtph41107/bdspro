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
	engine.Use(APIContexter())
	router := web.NewRouter(engine)
	cleanupCount := 0
	var first *APIContext
	var wrapped *typedIngressWriter
	router.AfterRouting(
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
	engine.Use(APIContexter())
	router := web.NewRouter(engine)

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

func TestTypedAPIContextCancellationPropagatesAndCleansScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(APIContexter())
	router := web.NewRouter(engine)

	requestContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	cleanupCount := 0
	router.Get("/cancel/:id", func(api *APIContext) {
		if api.Err() != nil {
			t.Errorf("request was already canceled: %v", api.Err())
		}
		if GetAPIContextFromRequest(api.CurrentRequest()) != api {
			t.Error("cancellation test received a different APIContext instance")
		}
		if id, err := api.PositivePathID("id"); err != nil || id != 42 {
			t.Errorf("matched route parameter id=%d err=%v", id, err)
		}
		store := reqctx.FromContext(api.CurrentRequest().Context())
		if store == nil {
			t.Fatal("request scope missing before cancellation")
		}
		store.AddCleanup(func() { cleanupCount++ })
		cancel()
		if api.Err() != context.Canceled {
			t.Errorf("APIContext.Err()=%v, want context.Canceled", api.Err())
		}
		select {
		case <-api.Done():
		default:
			t.Error("APIContext.Done() did not close after request cancellation")
		}
		api.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/cancel/42", nil).WithContext(requestContext)
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || cleanupCount != 1 {
		t.Fatalf("canceled request status=%d cleanup=%d, want 204 and exactly one cleanup", response.Code, cleanupCount)
	}
}

func TestTypedGinAbortWithoutResponseStopsNextAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(APIContexter())
	router := web.NewRouter(engine)

	cleanupCount := 0
	reached := false
	router.Get("/abort", append(
		GinActions(func(c *gin.Context) {
			store := reqctx.FromContext(c.Request.Context())
			if store == nil {
				t.Fatal("request scope was not initialized before Gin action")
			}
			store.AddCleanup(func() { cleanupCount++ })
			c.Abort()
			if c.Writer.Written() {
				t.Error("Abort alone unexpectedly committed an HTTP response")
			}
		}),
		func(*APIContext) { reached = true },
	)...)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/abort", nil))
	if reached || cleanupCount != 1 {
		t.Fatalf("aborted Gin action continued=%v, scope cleanup=%d, want false and 1", reached, cleanupCount)
	}
}

func TestTypedStatusSelectionWithoutAbortContinues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(APIContexter())
	router := web.NewRouter(engine)

	reached := false
	router.Get("/selected", func(api *APIContext) {
		api.Status(http.StatusAccepted)
		if api.WrittenStatus() != 0 {
			t.Error("selecting status should not commit an HTTP response")
		}
	}, func(api *APIContext) {
		reached = true
		api.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/selected", nil))
	if !reached || response.Code != http.StatusNoContent {
		t.Fatalf("uncommitted typed status stopped continuation: reached=%v status=%d", reached, response.Code)
	}
}

func TestBeforeRoutingAndTypedAPIContextShareOneCurrentRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(APIContexter())
	router := web.NewRouter(engine)
	router.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			scoped := req.WithContext(context.WithValue(req.Context(), typedIngressKey{}, "before-routing"))
			scoped.URL.Path = "/orgs/42"
			next.ServeHTTP(w, scoped)
		})
	})

	cleanupCount := 0
	router.Get("/orgs/:id", func(api *APIContext) {
		if api.Value(typedIngressKey{}) != "before-routing" {
			t.Errorf("APIContext lost BeforeRouting context value: %v", api.Value(typedIngressKey{}))
		}
		if api.CurrentRequest().URL.Path != "/orgs/42" {
			t.Errorf("APIContext observed stale ingress path %q", api.CurrentRequest().URL.Path)
		}
		if GetAPIContextFromRequest(api.CurrentRequest()) != api {
			t.Error("BeforeRouting created a second APIContext or lost the request-local provider")
		}
		if id, err := api.PositivePathID("id"); err != nil || id != 42 {
			t.Errorf("Gin route params after ingress rewrite: id=%d err=%v", id, err)
		}
		scope := reqctx.FromContext(api.CurrentRequest().Context())
		if scope == nil {
			t.Fatal("AfterRouting did not establish a request-local scope")
		}
		scope.AddCleanup(func() { cleanupCount++ })
		api.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	router.WrapIngress().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/incoming", nil))
	if w.Code != http.StatusNoContent || cleanupCount != 1 {
		t.Fatalf("ingress/typed context lifecycle status=%d cleanup=%d", w.Code, cleanupCount)
	}
}

// Recovery must not bypass request cleanup or retain the prior request's
// context. The same APIContexter owns scope finalization on success and panic.
func TestAPIContexterCleanupAfterGinRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery(), APIContexter())
	cleanupCount := 0
	var previous *APIContext
	engine.GET("/panic", func(c *gin.Context) {
		api := GetAPIContext(c)
		if previous == api {
			t.Error("APIContext was reused across recovered requests")
		}
		previous = api
		scope := reqctx.FromContext(api.CurrentRequest().Context())
		if scope == nil {
			t.Fatal("missing request-local data before action panic")
		}
		scope.AddCleanup(func() { cleanupCount++ })
		panic("request lifecycle panic fixture")
	})
	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))
		if response.Code != http.StatusInternalServerError || cleanupCount != i+1 {
			t.Fatalf("request %d: recovery status=%d cleanup=%d; want 500 and %d", i, response.Code, cleanupCount, i+1)
		}
	}
}
