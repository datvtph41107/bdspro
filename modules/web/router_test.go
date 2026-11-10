package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
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

type providerTestContext struct {
	transport *gin.Context
	status    int
}

func (c *providerTestContext) WrittenStatus() int { return c.status }

// Model the typed status provider separately from Gin. Gitea's web handler
// checks its ResponseStatusProvider and does not require a direct Gin import
// from a context implementation.
func TestTypedRouterUsesResponseStatusProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	reached := false
	router := NewRouter[*providerTestContext](engine.Group(""), func(c *gin.Context) *providerTestContext {
		if ctx, ok := c.Get("provider"); ok {
			return ctx.(*providerTestContext)
		}
		ctx := &providerTestContext{transport: c}
		c.Set("provider", ctx)
		return ctx
	})
	router.Get("/provider", func(ctx *providerTestContext) {
		ctx.transport.Status(http.StatusAccepted)
		ctx.status = http.StatusAccepted
	}, func(*providerTestContext) {
		reached = true
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/provider", nil))
	if reached || w.Code != http.StatusAccepted {
		t.Fatalf("provider did not terminate actions: reached=%v status=%d", reached, w.Code)
	}
}

// Merely choosing a status is not a committed response. A typed provider
// returning zero must not short-circuit later actions.
func TestTypedRouterProviderStatusOnlyContinues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	reached := false
	router := NewRouter[*providerTestContext](engine.Group(""), func(c *gin.Context) *providerTestContext {
		return &providerTestContext{transport: c}
	})
	router.Get("/status", func(ctx *providerTestContext) {
		ctx.transport.Status(http.StatusAccepted)
	}, func(ctx *providerTestContext) {
		reached = true
		ctx.transport.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/status", nil))
	if !reached || w.Code != http.StatusOK {
		t.Fatalf("status-only provider action interrupted: reached=%v status=%d", reached, w.Code)
	}
}

func TestTypedRouterRejectsMissingResponseStatusProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	defer func() {
		if recover() == nil {
			t.Fatal("invalid typed action context did not fail at route registration")
		}
	}()
	_ = NewRouter[*struct{}](engine.Group(""), func(*gin.Context) *struct{} { return &struct{}{} })
}

func TestAfterRoutingRunsAfterMatchBeforeGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	var calls []string
	m := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	m.AfterRouting(func(c *gin.Context) {
		if c.Param("id") != "42" {
			t.Fatalf("route params unavailable after matching: %q", c.Param("id"))
		}
		calls = append(calls, "after")
		c.Next()
	})
	m.Group("/orgs/:id", func() {
		m.Get("", func(c *gin.Context) {
			calls = append(calls, "action")
			c.Status(http.StatusNoContent)
		})
	}, func(*gin.Context) { calls = append(calls, "guard") })
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/not-an-api-route", nil))
	if len(calls) != 0 {
		t.Fatalf("post-route middleware ran for unmatched path: %v", calls)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orgs/42", nil))
	if w.Code != http.StatusNoContent || !reflect.DeepEqual(calls, []string{"after", "guard", "action"}) {
		t.Fatalf("post-route order=%v status=%d", calls, w.Code)
	}
}

func TestAfterRoutingRejectsNilMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewRouter[*gin.Context](gin.New().Group(""), func(c *gin.Context) *gin.Context { return c })
	defer func() {
		if recover() == nil {
			t.Fatal("nil post-routing middleware was accepted")
		}
	}()
	m.AfterRouting(nil)
}

func TestAfterRoutingSupportsTypedMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	m := NewRouter[*providerTestContext](engine.Group(""), func(c *gin.Context) *providerTestContext {
		if existing, ok := c.Get("context"); ok {
			return existing.(*providerTestContext)
		}
		value := &providerTestContext{transport: c}
		c.Set("context", value)
		return value
	})
	var calls []string
	m.AfterRouting(func(ctx *providerTestContext) {
		calls = append(calls, "after")
	}, func(c *gin.Context) {
		calls = append(calls, "gin")
		c.Next()
	})
	m.Get("/typed", func(*providerTestContext) {
		calls = append(calls, "action")
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/typed", nil))
	if w.Code != http.StatusOK || !reflect.DeepEqual(calls, []string{"after", "gin", "action"}) {
		t.Fatalf("typed post-routing middleware order=%v status=%d", calls, w.Code)
	}
}

func TestAfterRoutingRejectsUnsupportedSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewRouter[*gin.Context](gin.New().Group(""), func(c *gin.Context) *gin.Context { return c })
	defer func() {
		if recover() == nil {
			t.Fatal("unsupported post-routing signature was accepted")
		}
	}()
	m.AfterRouting(func(_ int) {})
}

func TestAfterRoutingTypedGinActionStopsAfterResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	m := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	hit := false
	m.AfterRouting(func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "denied"})
	})
	m.Get("/guarded", func(*gin.Context) { hit = true })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/guarded", nil))
	if hit || w.Code != http.StatusForbidden {
		t.Fatalf("typed post-route action continued: reached=%v status=%d", hit, w.Code)
	}
}

func TestMethodsRegistersMultipleHTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	r.Methods("GET, POST", "/multi", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(method, "/multi", nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("%s /multi status=%d", method, w.Code)
		}
	}
}

func TestMethodsRejectsEmptyHTTPMethodAtRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter[*gin.Context](gin.New().Group(""), func(c *gin.Context) *gin.Context { return c })
	defer func() {
		if recover() == nil {
			t.Fatal("empty HTTP method must be rejected during registration")
		}
	}()
	r.Methods("GET, ", "/invalid", func(*gin.Context) {})
}

type httpMiddlewareKey struct{}

func TestAfterRoutingNativeHTTPMiddlewarePassesRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var order []string
	r.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "http-before")
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), httpMiddlewareKey{}, "verified")))
			order = append(order, "http-after")
		})
	})
	r.Group("/teams", func() {
		r.Get("/:id", func(c *gin.Context) {
			order = append(order, "action")
			if c.Param("id") != "42" || c.Request.Context().Value(httpMiddlewareKey{}) != "verified" {
				t.Errorf("lost route parameter or HTTP context: id=%q", c.Param("id"))
			}
			c.Status(http.StatusNoContent)
		})
	}, func(*gin.Context) { order = append(order, "guard") })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/teams/42", nil))
	if w.Code != http.StatusNoContent ||
		!reflect.DeepEqual(order, []string{"http-before", "guard", "action", "http-after"}) {
		t.Fatalf("order=%v status=%d", order, w.Code)
	}
}

func TestAfterRoutingNativeHTTPMiddlewareStopsGinChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	hit := false
	r.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			// Intentionally do not call next.
		})
	})
	r.Get("/blocked", func(*gin.Context) { hit = true })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/blocked", nil))
	if hit || w.Code != http.StatusForbidden {
		t.Fatalf("HTTP middleware failed to short-circuit: hit=%v status=%d", hit, w.Code)
	}
}

func TestAfterRoutingHTTPHandlerFuncProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	r.AfterRouting(func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-HTTP-Provider", "yes")
			next.ServeHTTP(w, req)
		}
	})
	r.Get("/provider", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/provider", nil))
	if w.Code != http.StatusNoContent || w.Header().Get("X-HTTP-Provider") != "yes" {
		t.Fatalf("HTTP provider failed: status=%d header=%q", w.Code, w.Header().Get("X-HTTP-Provider"))
	}
}

func TestAfterRoutingRejectsNilHTTPProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter[*gin.Context](gin.New().Group(""), func(c *gin.Context) *gin.Context { return c })
	var absent func(http.Handler) http.Handler
	defer func() {
		if recover() == nil {
			t.Fatal("nil HTTP middleware was accepted")
		}
	}()
	r.AfterRouting(absent)
}

func TestAfterRoutingHTTPActionContinuesWhenResponseNotWritten(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var steps []string
	r.AfterRouting(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		steps = append(steps, "http")
		w.Header().Set("X-Action", "continued")
	}))
	r.Get("/continue", func(c *gin.Context) {
		steps = append(steps, "gin")
		c.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/continue", nil))
	if w.Code != http.StatusNoContent || w.Header().Get("X-Action") != "continued" ||
		!reflect.DeepEqual(steps, []string{"http", "gin"}) {
		t.Fatalf("HTTP action did not continue: steps=%v status=%d", steps, w.Code)
	}
}

func TestAfterRoutingHTTPActionStopsAfterWritingResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	hit := false
	r.AfterRouting(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	r.Get("/denied", func(*gin.Context) { hit = true })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/denied", nil))
	if hit || w.Code != http.StatusForbidden {
		t.Fatalf("HTTP action leaked into Gin actions: hit=%v status=%d", hit, w.Code)
	}
}

func TestAfterRoutingHTTPActionWriteImplicitlyCommitsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	hit := false
	r.AfterRouting(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("done"))
	}))
	r.Get("/body", func(*gin.Context) { hit = true })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/body", nil))
	if hit || w.Code != http.StatusOK || w.Body.String() != "done" {
		t.Fatalf("HTTP body write did not stop execution: hit=%v status=%d body=%q", hit, w.Code, w.Body.String())
	}
}

func TestAfterRoutingPlainHTTPFunctionUsesCommonProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	hit := false
	r.AfterRouting(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	r.Get("/plain", func(*gin.Context) { hit = true })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/plain", nil))
	if hit || w.Code != http.StatusUnauthorized {
		t.Fatalf("raw HTTP action did not stop Gin chain: hit=%v status=%d", hit, w.Code)
	}
}

type recordingHTTPWriter struct {
	http.ResponseWriter
	writtenBytes int
	status       int
}

func (w *recordingHTTPWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *recordingHTTPWriter) Write(body []byte) (int, error) {
	n, err := w.ResponseWriter.Write(body)
	w.writtenBytes += n
	return n, err
}

func TestAfterRoutingPreservesWrappedHTTPResponseWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var observed *recordingHTTPWriter
	r.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			observed = &recordingHTTPWriter{ResponseWriter: w}
			next.ServeHTTP(observed, req)
		})
	})
	r.Get("/wrapped", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/wrapped", nil))
	if observed == nil || observed.status != http.StatusCreated ||
		observed.writtenBytes == 0 || w.Code != http.StatusCreated {
		t.Fatalf("Gin bypassed HTTP wrapper: observed=%+v code=%d", observed, w.Code)
	}
}

type typedHTTPContextKey struct{}

type typedHTTPContext struct {
	writer http.ResponseWriter
	status int
}

func (ctx *typedHTTPContext) WrittenStatus() int { return ctx.status }

func init() {
	RegisterResponseStatusProvider[*typedHTTPContext](func(req *http.Request) web_types.ResponseStatusProvider {
		return req.Context().Value(typedHTTPContextKey{}).(*typedHTTPContext)
	})
}

func TestAfterRoutingRegisteredTypedHTTPContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	hit := false
	r.AfterRouting(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := &typedHTTPContext{writer: w}
				next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), typedHTTPContextKey{}, ctx)))
			})
		},
		func(ctx *typedHTTPContext) {
			ctx.status = http.StatusForbidden
			ctx.writer.WriteHeader(ctx.status)
		},
	)
	r.Get("/typed-http", func(*gin.Context) { hit = true })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/typed-http", nil))
	if hit || w.Code != http.StatusForbidden {
		t.Fatalf("typed HTTP provider did not terminate chain: hit=%v status=%d", hit, w.Code)
	}
}

func TestRegisteredTypedHandlerRejectsReturnValueAtRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter[*gin.Context](gin.New().Group(""), func(c *gin.Context) *gin.Context { return c })
	defer func() {
		if recover() == nil {
			t.Fatal("typed handler with a return value was accepted")
		}
	}()
	r.AfterRouting(func(*typedHTTPContext) error { return nil })
}

func TestRegisteredTypedHandlerWithoutWriteContinues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var steps []string
	r.AfterRouting(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := &typedHTTPContext{writer: w}
				next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), typedHTTPContextKey{}, ctx)))
			})
		},
		func(*typedHTTPContext) { steps = append(steps, "typed") },
	)
	r.Get("/continue-typed", func(c *gin.Context) {
		steps = append(steps, "endpoint")
		c.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/continue-typed", nil))
	if w.Code != http.StatusNoContent || !reflect.DeepEqual(steps, []string{"typed", "endpoint"}) {
		t.Fatalf("typed HTTP action did not continue: steps=%v status=%d", steps, w.Code)
	}
}

func TestRegisteredTypedRouterAcceptsPreMiddlewareInGroupAndMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	var steps []string
	r := NewRouter[*typedHTTPContext](engine.Group(""), func(c *gin.Context) *typedHTTPContext {
		return c.Request.Context().Value(typedHTTPContextKey{}).(*typedHTTPContext)
	})
	r.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := &typedHTTPContext{writer: w}
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), typedHTTPContextKey{}, ctx)))
		})
	})
	trace := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				steps = append(steps, name)
				next.ServeHTTP(w, req)
			})
		}
	}
	r.Group("/teams", func() {
		r.Post("", web_types.PreMiddlewareProvider(trace("route-pre")),
			trace("route-normal"),
			func(ctx *typedHTTPContext) {
				steps = append(steps, "action")
				ctx.writer.WriteHeader(http.StatusCreated)
			})
	}, trace("group-normal"), web_types.PreMiddlewareProvider(trace("group-pre")))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/teams", nil))
	want := []string{"group-pre", "route-pre", "group-normal", "route-normal", "action"}
	if w.Code != http.StatusCreated || !reflect.DeepEqual(steps, want) {
		t.Fatalf("status=%d order=%v want=%v", w.Code, steps, want)
	}
}

func TestRegisteredRouterAfterRoutingPreMiddlewareRunsBeforeNormalScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	var calls []string
	m := NewRouter[*typedHTTPContext](engine.Group(""), func(c *gin.Context) *typedHTTPContext {
		return c.Request.Context().Value(typedHTTPContextKey{}).(*typedHTTPContext)
	})
	trace := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls = append(calls, name)
				next.ServeHTTP(w, req)
			})
		}
	}
	m.AfterRouting(
		trace("normal-after"),
		web_types.PreMiddlewareProvider(trace("pre-after")),
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := &typedHTTPContext{writer: w}
				next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), typedHTTPContextKey{}, ctx)))
			})
		},
	)
	m.Group("/teams", func() {
		m.Get("/:id",
			trace("normal-route"),
			web_types.PreMiddlewareProvider(trace("pre-route")),
			func(ctx *typedHTTPContext) {
				calls = append(calls, "endpoint")
				ctx.writer.WriteHeader(http.StatusNoContent)
			},
		)
	}, trace("normal-group"), web_types.PreMiddlewareProvider(trace("pre-group")))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/teams/42", nil))
	want := []string{
		"pre-after", "pre-group", "pre-route",
		"normal-after", "normal-group", "normal-route", "endpoint",
	}
	if w.Code != http.StatusNoContent || !reflect.DeepEqual(calls, want) {
		t.Fatalf("status=%d calls=%v want=%v", w.Code, calls, want)
	}
}

func TestAfterRoutingRegisteredTypedHandlerRejectsBadSignatureEarly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewRouter[*typedHTTPContext](gin.New().Group(""), func(c *gin.Context) *typedHTTPContext {
		return c.Request.Context().Value(typedHTTPContextKey{}).(*typedHTTPContext)
	})
	defer func() {
		if recover() == nil {
			t.Fatal("invalid typed handler accepted at AfterRouting registration")
		}
	}()
	m.AfterRouting(func(*typedHTTPContext) error { return nil })
}

func TestRegisteredTypedRouterRejectsNilSecurityGuardsAtRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name     string
		register func(*Router[*typedHTTPContext])
	}{
		{
			name: "nil group guard",
			register: func(r *Router[*typedHTTPContext]) {
				r.Group("/private", func() {
					r.Get("", func(*typedHTTPContext) {})
				}, nil)
			},
		},
		{
			name: "nil typed group guard",
			register: func(r *Router[*typedHTTPContext]) {
				var guard func(*typedHTTPContext)
				r.Group("/private", func() {
					r.Get("", func(*typedHTTPContext) {})
				}, guard)
			},
		},
		{
			name: "nil pre group guard",
			register: func(r *Router[*typedHTTPContext]) {
				r.Group("/private", func() {
					r.Get("", func(*typedHTTPContext) {})
				}, web_types.PreMiddlewareProvider(nil))
			},
		},
		{
			name: "nil route guard",
			register: func(r *Router[*typedHTTPContext]) {
				r.Get("/private", nil, func(*typedHTTPContext) {})
			},
		},
		{
			name: "nil typed route guard",
			register: func(r *Router[*typedHTTPContext]) {
				var guard func(*typedHTTPContext)
				r.Get("/private", guard, func(*typedHTTPContext) {})
			},
		},
		{
			name: "nil pre route guard",
			register: func(r *Router[*typedHTTPContext]) {
				r.Get("/private", web_types.PreMiddlewareProvider(nil), func(*typedHTTPContext) {})
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			r := NewRouter[*typedHTTPContext](engine.Group(""), func(c *gin.Context) *typedHTTPContext {
				return c.Request.Context().Value(typedHTTPContextKey{}).(*typedHTTPContext)
			})
			defer func() {
				if recover() == nil {
					t.Fatal("nil security guard was silently omitted during route registration")
				}
			}()
			tc.register(r)
		})
	}
}

func TestRegisteredTypedRouterRejectsDuplicateHTTPContinuation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*typedHTTPContext](engine.Group(""), func(c *gin.Context) *typedHTTPContext {
		return c.Request.Context().Value(typedHTTPContextKey{}).(*typedHTTPContext)
	})
	r.AfterRouting(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := &typedHTTPContext{writer: w}
				request := req.WithContext(context.WithValue(req.Context(), typedHTTPContextKey{}, ctx))
				next.ServeHTTP(w, request)
				next.ServeHTTP(w, request)
			})
		},
	)
	calls := 0
	r.Get("/duplicate-next", func(*typedHTTPContext) {
		calls++
	})
	defer func() {
		if recover() == nil {
			t.Error("middleware calling next twice was accepted")
		}
		if calls != 1 {
			t.Errorf("endpoint executed %d times, want once before duplicate-next rejection", calls)
		}
	}()
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/duplicate-next", nil))
}

type ingressTraceKey struct{}

func TestBeforeRoutingRunsBeforeGinMatchingIncluding404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var events []string
	r.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			events = append(events, "before")
			next.ServeHTTP(w, req)
			events = append(events, "after")
		})
	})
	engine.NoRoute(func(c *gin.Context) {
		events = append(events, "not-found")
		c.Status(http.StatusNotFound)
	})
	r.Get("/exists", func(c *gin.Context) {
		events = append(events, "matched")
		c.Status(http.StatusNoContent)
	})
	ingress := r.WrapIngress(engine)
	for _, tc := range []struct {
		path   string
		status int
		events []string
	}{
		{"/missing", http.StatusNotFound, []string{"before", "not-found", "after"}},
		{"/exists", http.StatusNoContent, []string{"before", "matched", "after"}},
	} {
		events = nil
		rec := httptest.NewRecorder()
		ingress.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.status || !reflect.DeepEqual(events, tc.events) {
			t.Errorf("%s: status=%d events=%v, want status=%d events=%v", tc.path, rec.Code, events, tc.status, tc.events)
		}
	}
}

func TestBeforeRoutingRewritesMethodAndPathBeforeGinAndPreservesWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var observed *recordingHTTPWriter
	r.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			observed = &recordingHTTPWriter{ResponseWriter: w}
			rewritten := req.WithContext(context.WithValue(req.Context(), ingressTraceKey{}, "ingress"))
			rewritten.Method = http.MethodPost
			rewritten.URL.Path = "/target"
			next.ServeHTTP(observed, rewritten)
		})
	})
	r.Post("/target", func(c *gin.Context) {
		if c.Request.Method != http.MethodPost || c.Request.URL.Path != "/target" || c.Request.Context().Value(ingressTraceKey{}) != "ingress" {
			t.Errorf("Gin matcher did not receive ingress method/path/context: method=%s path=%s", c.Request.Method, c.Request.URL.Path)
		}
		c.JSON(http.StatusAccepted, gin.H{"ok": true})
	})
	rec := httptest.NewRecorder()
	r.WrapIngress(engine).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/source", nil))
	if rec.Code != http.StatusAccepted || observed == nil || observed.status != http.StatusAccepted || observed.writtenBytes == 0 {
		t.Fatalf("ingress rewrite/writer status=%d observed=%+v", rec.Code, observed)
	}
}

func TestBeforeRoutingMayStopRequestWithoutMatchingAndStillHandleHEAD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	enteredMatcher := false
	beforeCalls := 0
	engine.NoRoute(func(c *gin.Context) {
		enteredMatcher = true
		c.Status(http.StatusNotFound)
	})
	r.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			beforeCalls++
			if req.URL.Path == "/blocked" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Head("/head", func(c *gin.Context) {
		enteredMatcher = true
		c.Status(http.StatusNoContent)
	})
	ingress := r.WrapIngress(engine)
	blocked := httptest.NewRecorder()
	ingress.ServeHTTP(blocked, httptest.NewRequest(http.MethodGet, "/blocked", nil))
	if blocked.Code != http.StatusForbidden || enteredMatcher || beforeCalls != 1 {
		t.Fatalf("blocked: status=%d entered=%v beforeCalls=%d", blocked.Code, enteredMatcher, beforeCalls)
	}
	head := httptest.NewRecorder()
	ingress.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/head", nil))
	if head.Code != http.StatusNoContent || !enteredMatcher || beforeCalls != 2 {
		t.Fatalf("HEAD: status=%d entered=%v beforeCalls=%d", head.Code, enteredMatcher, beforeCalls)
	}
}

func TestBeforeRoutingRejectsInvalidAndLateRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		action any
	}{
		{"untyped nil", nil},
		{"typed nil", (func(http.Handler) http.Handler)(nil)},
		{"nil pre provider", web_types.PreMiddlewareProvider(nil)},
		{"Gin-only handler", gin.HandlerFunc(func(*gin.Context) {})},
		{"typed API action", func(*typedHTTPContext) {}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
			defer func() {
				if recover() == nil {
					t.Error("invalid pre-matcher middleware was accepted")
				}
			}()
			r.BeforeRouting(tc.action)
		})
	}
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	r.WrapIngress(engine)
	defer func() {
		if recover() == nil {
			t.Error("mutating ingress middleware after composition was accepted")
		}
	}()
	r.BeforeRouting(func(next http.Handler) http.Handler { return next })
}

func TestBeforeRoutingRejectsDuplicateContinuation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	r.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req)
			next.ServeHTTP(w, req)
		})
	})
	reached := 0
	r.Get("/once", func(c *gin.Context) {
		reached++
		c.Status(http.StatusNoContent)
	})
	defer func() {
		if recover() == nil {
			t.Error("BeforeRouting allowed the same request to enter Gin twice")
		}
		if reached != 1 {
			t.Errorf("Gin endpoint executed %d times; want exactly once", reached)
		}
	}()
	r.WrapIngress(engine).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/once", nil))
}

func TestBeforeRoutingCompilesStatefulProviderOnceAcrossRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	compiled := 0
	entered := 0
	r.BeforeRouting(func(next http.Handler) http.Handler {
		compiled++
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			entered++
			next.ServeHTTP(w, req)
		})
	})
	actions := 0
	r.Get("/repeat", func(c *gin.Context) {
		actions++
		c.Status(http.StatusNoContent)
	})
	ingress := r.WrapIngress(engine)
	if compiled != 1 {
		t.Fatalf("provider compiled %d times during ingress setup, want once", compiled)
	}
	for i := 0; i < 3; i++ {
		response := httptest.NewRecorder()
		ingress.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/repeat", nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("request %d status=%d, want 204", i, response.Code)
		}
	}
	if compiled != 1 || entered != 3 || actions != 3 {
		t.Fatalf("compiled=%d entered=%d actions=%d, want 1/3/3", compiled, entered, actions)
	}
}

// A bad second HTTP method must never leave the first one registered. Route
// setup is startup-only: partially registered routes violate fail-closed setup.
func TestMethodsRejectsPartialRegistrationWhenAnyMethodInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, methods := range []string{"GET, ", "GET,GET", " ,POST", "GET,geT", "GET,GET?", "GET,GÉT"} {
		t.Run(methods, func(t *testing.T) {
			engine := gin.New()
			r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("expected panic for invalid methods %q", methods)
					}
				}()
				r.Methods(methods, "/no-partial", func(c *gin.Context) {
					c.Status(http.StatusNoContent)
				})
			}()
			if routes := engine.Routes(); len(routes) != 0 {
				t.Fatalf("methods %q registered %v before validation completed", methods, routes)
			}
		})
	}
}

func TestBeforeRoutingRunsOnMethodNotAllowedWithGin405Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	// Gin.New does not enable 405 routing by default. The coordinator
	// decides this production policy at the actual engine entrypoint.
	engine.HandleMethodNotAllowed = true
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var steps []string
	r.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			steps = append(steps, "before")
			next.ServeHTTP(w, req)
			steps = append(steps, "after")
		})
	})
	engine.NoMethod(func(c *gin.Context) {
		steps = append(steps, "method-not-allowed")
		c.Status(http.StatusMethodNotAllowed)
	})
	r.Get("/read-only", func(c *gin.Context) {
		steps = append(steps, "endpoint")
		c.Status(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	r.WrapIngress(engine).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/read-only", nil))
	want := []string{"before", "method-not-allowed", "after"}
	if rec.Code != http.StatusMethodNotAllowed || !reflect.DeepEqual(steps, want) {
		t.Fatalf("405 routing: status=%d steps=%v, want %d %v", rec.Code, steps, http.StatusMethodNotAllowed, want)
	}
}

func TestBeforeRoutingCannotRegisterAfterRouteMatchingWasConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	r.Get("/already-registered", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	defer func() {
		if recover() == nil {
			t.Fatal("BeforeRouting was accepted after route registration")
		}
		if len(r.beforeRouting) != 0 {
			t.Fatalf("late BeforeRouting changed configured ingress: %d handlers", len(r.beforeRouting))
		}
	}()
	r.BeforeRouting(func(next http.Handler) http.Handler { return next })
}

// Middleware registration is a startup transaction: invalid input must not
// leave earlier entries from the same call behind after a recovered panic.
func TestBeforeRoutingBatchRegistrationIsAtomic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	valid := func(next http.Handler) http.Handler { return next }

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("invalid second BeforeRouting middleware must panic")
			}
		}()
		r.BeforeRouting(valid, func(*gin.Context) {})
	}()
	if len(r.beforeRouting) != 0 {
		t.Fatalf("failed BeforeRouting call retained %d providers", len(r.beforeRouting))
	}

	r.BeforeRouting(valid)
	r.Get("/healthy", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	r.WrapIngress(engine).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthy", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("retry after invalid middleware returned %d, want 204", w.Code)
	}
}

func TestAfterRoutingBatchRegistrationIsAtomic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var calls []string
	validGin := func(c *gin.Context) { calls = append(calls, "gin"); c.Next() }
	validHTTP := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			calls = append(calls, "http")
			next.ServeHTTP(w, req)
		})
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("invalid third AfterRouting middleware must panic")
			}
		}()
		r.AfterRouting(validGin, validHTTP, struct{}{})
	}()
	if len(r.afterRouting) != 0 || len(r.afterRoutingHTTP) != 0 {
		t.Fatalf("failed AfterRouting call retained Gin=%d HTTP=%d middleware", len(r.afterRouting), len(r.afterRoutingHTTP))
	}

	r.AfterRouting(validGin, validHTTP)
	r.Get("/healthy", func(c *gin.Context) {
		calls = append(calls, "action")
		c.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthy", nil))
	want := []string{"gin", "http", "action"}
	if w.Code != http.StatusNoContent || !reflect.DeepEqual(calls, want) {
		t.Fatalf("status=%d calls=%v, want 204 and %v", w.Code, calls, want)
	}
}

// HTTP middleware must not resume a completed request. In addition to
// duplicate-next protection, the ingress must reject a first late next:
// the original ResponseWriter and request-local data are no longer live.
func TestBeforeRoutingRejectsContinuationAfterRequestCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*gin.Context](engine.Group(""), func(c *gin.Context) *gin.Context { return c })
	var resume func()
	reached := false
	r.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			resume = func() { next.ServeHTTP(w, req) }
			w.WriteHeader(http.StatusAccepted)
		})
	})
	r.Get("/late", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	r.WrapIngress(engine).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/late", nil))
	if w.Code != http.StatusAccepted || reached || resume == nil {
		t.Fatalf("initial request status=%d reached=%v resume-nil=%v", w.Code, reached, resume == nil)
	}
	defer func() {
		if recover() == nil {
			t.Error("continuation entered the matcher after request completion")
		}
		if reached {
			t.Error("late continuation reached the completed request's endpoint")
		}
	}()
	resume()
}

// As in Gitea/Chi, middleware provider factories are compiled during route
// registration. They must not be reconstructed for each matched request:
// doing so resets middleware-local rate limiters, counters, or caches.
func TestRegisteredHTTPMiddlewareProviderCompiledOnceForRepeatedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*typedHTTPContext](engine.Group(""), func(c *gin.Context) *typedHTTPContext {
		return c.Request.Context().Value(typedHTTPContextKey{}).(*typedHTTPContext)
	})
	factoryCalls, requests, completed := 0, 0, 0
	r.AfterRouting(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := &typedHTTPContext{writer: w}
				next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), typedHTTPContextKey{}, ctx)))
			})
		},
		func(next http.Handler) http.Handler {
			factoryCalls++
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests++
				next.ServeHTTP(w, req)
			})
		},
	)
	r.Get("/stateful", func(ctx *typedHTTPContext) {
		completed++
		ctx.writer.WriteHeader(http.StatusNoContent)
	})
	if factoryCalls != 1 {
		t.Fatalf("provider factory compiled %d times at registration, want 1", factoryCalls)
	}
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stateful", nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("request %d returned %d, want 204", i, w.Code)
		}
	}
	if factoryCalls != 1 || requests != 3 || completed != 3 {
		t.Fatalf("factories=%d requests=%d completed=%d, want 1/3/3", factoryCalls, requests, completed)
	}
}

// Middleware factories live for the route's lifetime, but their continuations
// must remain request-local. A saved next must not re-enter a completed Gin
// request, even if the middleware never called it during that request.
func TestRegisteredTypedRouteRejectsContinuationAfterCompletion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter[*typedHTTPContext](engine.Group(""), func(c *gin.Context) *typedHTTPContext {
		return c.Request.Context().Value(typedHTTPContextKey{}).(*typedHTTPContext)
	})
	var resume func()
	reached := false
	r.AfterRouting(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				resume = func() { next.ServeHTTP(w, req) }
				w.WriteHeader(http.StatusAccepted)
			})
		},
	)
	r.Get("/expired-next", func(*typedHTTPContext) { reached = true })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/expired-next", nil))
	if rec.Code != http.StatusAccepted || reached || resume == nil {
		t.Fatalf("initial status=%d reached=%v next-set=%v", rec.Code, reached, resume != nil)
	}
	defer func() {
		if recover() == nil {
			t.Error("compiled HTTP chain accepted continuation after request returned")
		}
		if reached {
			t.Error("expired continuation reached the typed route endpoint")
		}
	}()
	resume()
}
