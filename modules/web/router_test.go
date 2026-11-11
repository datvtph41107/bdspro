package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"github.com/gin-gonic/gin"
)

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

func TestRegisteredTypedRouterAcceptsPreMiddlewareInGroupAndMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	var steps []string
	r := NewRouter(engine)
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
	m := NewRouter(engine)
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
	m := NewRouter(gin.New())
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
		register func(*Router)
	}{
		{
			name: "nil group guard",
			register: func(r *Router) {
				r.Group("/private", func() {
					r.Get("", func(*typedHTTPContext) {})
				}, nil)
			},
		},
		{
			name: "nil typed group guard",
			register: func(r *Router) {
				var guard func(*typedHTTPContext)
				r.Group("/private", func() {
					r.Get("", func(*typedHTTPContext) {})
				}, guard)
			},
		},
		{
			name: "nil pre group guard",
			register: func(r *Router) {
				r.Group("/private", func() {
					r.Get("", func(*typedHTTPContext) {})
				}, web_types.PreMiddlewareProvider(nil))
			},
		},
		{
			name: "nil route guard",
			register: func(r *Router) {
				r.Get("/private", nil, func(*typedHTTPContext) {})
			},
		},
		{
			name: "nil typed route guard",
			register: func(r *Router) {
				var guard func(*typedHTTPContext)
				r.Get("/private", guard, func(*typedHTTPContext) {})
			},
		},
		{
			name: "nil pre route guard",
			register: func(r *Router) {
				r.Get("/private", web_types.PreMiddlewareProvider(nil), func(*typedHTTPContext) {})
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			r := NewRouter(engine)
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
	r := NewRouter(engine)
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

func TestRegisteredHTTPMiddlewareProviderCompiledOnceForRepeatedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter(engine)
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
	r := NewRouter(engine)
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

// One route-level compiled chain must safely serve independent requests.
// The middleware factories run once at route registration; their request
// forwarding state and duplicate-next guards may never be shared.

func TestMatchedRouteCompiledProvidersRemainRequestLocalUnderConcurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter(engine)
	var factoryCount, middlewareCalls, endpointCalls atomic.Int32
	type traceKey struct{}
	r.AfterRouting(
		func(next http.Handler) http.Handler {
			factoryCount.Add(1)
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				middlewareCalls.Add(1)
				id := req.Header.Get("X-Request-ID")
				req = req.WithContext(context.WithValue(req.Context(), traceKey{}, id))
				ctx := &typedHTTPContext{writer: w}
				req = req.WithContext(context.WithValue(req.Context(), typedHTTPContextKey{}, ctx))
				next.ServeHTTP(w, req)
			})
		},
		func(next http.Handler) http.Handler {
			factoryCount.Add(1)
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				middlewareCalls.Add(1)
				if req.Context().Value(traceKey{}) != req.Header.Get("X-Request-ID") {
					t.Errorf("request-local context leaked or disappeared for %q", req.Header.Get("X-Request-ID"))
				}
				next.ServeHTTP(w, req)
			})
		},
	)
	r.Get("/concurrent", func(ctx *typedHTTPContext) {
		endpointCalls.Add(1)
		ctx.writer.WriteHeader(http.StatusNoContent)
	})
	if got := factoryCount.Load(); got != 2 {
		t.Fatalf("compiled middleware factories=%d, want 2", got)
	}
	const total = 24
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/concurrent", nil)
			req.Header.Set("X-Request-ID", strconv.Itoa(i))
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			if w.Code != http.StatusNoContent {
				t.Errorf("request %d status %d, want 204", i, w.Code)
			}
		}(i)
	}
	wg.Wait()
	if got := factoryCount.Load(); got != 2 {
		t.Errorf("stateful provider was recompiled: factories=%d, want 2", got)
	}
	if got := middlewareCalls.Load(); got != 2*total {
		t.Errorf("middleware calls=%d, want %d", got, 2*total)
	}
	if got := endpointCalls.Load(); got != total {
		t.Errorf("endpoint calls=%d, want %d", got, total)
	}
}

func TestBeforeRoutingRunsBeforeMatchingAndHandles404And405(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	r := NewRouter(engine)
	var order []string
	r.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "before")
			next.ServeHTTP(w, req)
			order = append(order, "after")
		})
	})
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		order = append(order, "endpoint")
		w.WriteHeader(http.StatusNoContent)
	})
	handler := r.WrapIngress()
	for _, tc := range []struct {
		method string
		path   string
		status int
		want   []string
	}{
		{http.MethodGet, "/health", http.StatusNoContent, []string{"before", "endpoint", "after"}},
		{http.MethodGet, "/missing", http.StatusNotFound, []string{"before", "after"}},
		{http.MethodPost, "/health", http.StatusMethodNotAllowed, []string{"before", "after"}},
	} {
		order = nil
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status || !reflect.DeepEqual(order, tc.want) {
			t.Errorf("%s %s: status=%d steps=%v want %d %v", tc.method, tc.path, w.Code, order, tc.status, tc.want)
		}
	}
}

func TestMethodsRejectInvalidMethodBatchBeforeRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter(engine)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("bad method list did not panic")
			}
		}()
		r.Methods("GET,geT", "/route", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	}()
	if got := len(engine.Routes()); got != 0 {
		t.Fatalf("invalid method batch registered %d routes", got)
	}
}

func TestBeforeRoutingRejectsLateContinuation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter(engine)
	var resume func()
	reached := false
	r.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			resume = func() { next.ServeHTTP(w, req) }
			w.WriteHeader(http.StatusAccepted)
		})
	})
	r.Get("/late", func(w http.ResponseWriter, req *http.Request) {
		reached = true
	})
	w := httptest.NewRecorder()
	r.WrapIngress().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/late", nil))
	if w.Code != http.StatusAccepted || reached || resume == nil {
		t.Fatalf("initial status=%d reached=%v continuation-set=%v", w.Code, reached, resume != nil)
	}
	defer func() {
		if recover() == nil {
			t.Error("late continuation accepted")
		}
		if reached {
			t.Error("late continuation reached completed endpoint")
		}
	}()
	resume()
}

func TestAfterRoutingRejectsRegistrationAfterCompiledRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	r := NewRouter(engine)
	r.AfterRouting(func(next http.Handler) http.Handler { return next })
	r.Get("/before", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	defer func() {
		if recover() == nil {
			t.Fatal("late AfterRouting registration must panic instead of silently omitting earlier routes")
		}
		if got := len(engine.Routes()); got != 1 {
			t.Fatalf("late AfterRouting changed route inventory: got %d, want 1", got)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/before", nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("previously compiled route changed status: got %d", w.Code)
		}
	}()

	r.AfterRouting(func(next http.Handler) http.Handler { return next })
}
