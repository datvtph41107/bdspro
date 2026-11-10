package web

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"github.com/gin-gonic/gin"
)

// ingressContinuationState belongs to one HTTP request, not to a compiled
// middleware provider. An atomic per-provider flag prevents duplicate matcher
// execution even if a defective middleware calls next concurrently.
type ingressContinuationKey struct{}

type ingressContinuationState struct {
	called   []atomic.Bool
	finished atomic.Bool
}

func isNilOrFuncNil(handler any) bool {
	if handler == nil {
		return true
	}
	value := reflect.ValueOf(handler)
	return value.Kind() == reflect.Func && value.IsNil()
}

func wrapMiddlewareAppendPre(all []middlewareProvider, handlers []any) []middlewareProvider {
	for _, handler := range handlers {
		if pre, ok := handler.(web_types.PreMiddlewareProvider); ok {
			if pre == nil {
				panic("web: nil pre middleware")
			}
			all = append(all, toHandlerProvider(middlewareProvider(pre)))
		}
	}
	return all
}

func wrapMiddlewareAppendNormal(all []middlewareProvider, handlers []any) []middlewareProvider {
	for _, handler := range handlers {
		if _, pre := handler.(web_types.PreMiddlewareProvider); pre {
			continue
		}
		if isNilOrFuncNil(handler) {
			panic("web: nil middleware or route action")
		}
		all = append(all, toHandlerProvider(handler))
	}
	return all
}

// wrapMiddlewareAndHandler is the single matched-route HTTP compiler.
// Pre providers run ahead of ordinary middleware from the same scopes;
// the final non-nil action is the endpoint and cannot be a pre middleware.
func wrapMiddlewareAndHandler(useMiddlewares, curMiddlewares, handlers []any) ([]middlewareProvider, http.HandlerFunc, bool) {
	if len(handlers) == 0 {
		panic("web: no endpoint handler provided")
	}
	if isNilOrFuncNil(handlers[len(handlers)-1]) {
		panic("web: endpoint handler cannot be nil")
	}
	if _, pre := handlers[len(handlers)-1].(web_types.PreMiddlewareProvider); pre {
		panic("web: endpoint cannot be pre middleware")
	}
	providers := make([]middlewareProvider, 0, len(useMiddlewares)+len(curMiddlewares)+len(handlers))
	providers = wrapMiddlewareAppendPre(providers, useMiddlewares)
	providers = wrapMiddlewareAppendPre(providers, curMiddlewares)
	providers = wrapMiddlewareAppendPre(providers, handlers)
	hasPre := len(providers) != 0
	providers = wrapMiddlewareAppendNormal(providers, useMiddlewares)
	providers = wrapMiddlewareAppendNormal(providers, curMiddlewares)
	providers = wrapMiddlewareAppendNormal(providers, handlers)
	if len(providers) == 0 {
		panic("web: no compilable endpoint handler")
	}
	last := len(providers) - 1
	return providers[:last], providers[last](nil).ServeHTTP, hasPre
}

// FormStore owns request-local data through reqctx; web does not manage a
// separate registry or a global cache. The form is exposed only after binding.
type FormStore = reqctx.ContextDataProvider

const formKey = "__form"

func SetForm(store FormStore, value any) {
	store.GetData()[formKey] = value
}

func GetForm[T any](store FormStore) T {
	value := store.GetData()[formKey]
	form, ok := value.(T)
	if !ok {
		panic(fmt.Errorf("web: bound form %T does not match requested %s", value, reflect.TypeFor[T]()))
	}
	return form
}

// Router owns one Gin matcher and one Gitea-style HTTP handler compiler.
type Router struct {
	engine           *gin.Engine
	group            *gin.RouterGroup
	beforeRouting    []middlewareProvider
	ingressBound     bool
	routesRegistered bool
	afterRouting     []any
	curGroupPrefix   string
	curMiddlewares   []any
}

func NewRouter(engine *gin.Engine) *Router {
	if engine == nil {
		panic("web: nil Gin engine")
	}
	return &Router{engine: engine, group: engine.Group("")}
}

// BeforeRouting installs HTTP-native middleware ahead of Gin's route matcher.
// Gin's Engine.Use runs after route lookup, so adding a matched-route handler
// here would give incorrect semantics for unmatched routes and method rewrites.
// Only HTTP-native handlers are permitted: APIContext and Gin route parameters
// do not exist until the engine selects the route.
func (r *Router) BeforeRouting(middlewares ...any) {
	if r.ingressBound {
		panic("web: cannot register BeforeRouting after ingress composition")
	}
	if r.routesRegistered {
		panic("web: BeforeRouting middleware must be declared before routes")
	}
	// Compile the entire batch before mutating router configuration.
	// A recovered startup panic must not leave a partially installed chain.
	compiled := make([]middlewareProvider, 0, len(middlewares))
	for _, middleware := range middlewares {
		if isNilOrFuncNil(middleware) {
			panic("web: nil before-routing middleware")
		}
		if pre, ok := middleware.(web_types.PreMiddlewareProvider); ok {
			compiled = append(compiled, toHandlerProvider(middlewareProvider(pre)))
			continue
		}
		switch middleware.(type) {
		case middlewareProvider, func(http.Handler) http.HandlerFunc,
			http.HandlerFunc, func(http.ResponseWriter, *http.Request):
			compiled = append(compiled, toHandlerProvider(middleware))
		default:
			panic(fmt.Sprintf("web: BeforeRouting requires an HTTP-native middleware, got %T", middleware))
		}
	}
	r.beforeRouting = append(r.beforeRouting, compiled...)
}

// WrapIngress composes BeforeRouting middleware around the SAME Gin engine
// used by Methods, without creating another matcher or middleware executor.
// The application HTTP entrypoint must serve the returned handler: simply
// invoking BeforeRouting without wrapping the engine does not activate it.
// Unlike AfterRouting, this outer chain also executes for Gin 404/405 routes.
func (r *Router) WrapIngress() http.Handler {
	engine := http.Handler(r.engine)
	if r.ingressBound {
		panic("web: ingress has already been composed")
	}
	r.ingressBound = true
	// Compile providers once, as in Gitea. Continuation guards use a
	// request-local state so stateful providers are never reconstructed on
	// every request (which would reset rate limiters or other state).
	handler := engine
	for i := len(r.beforeRouting) - 1; i >= 0; i-- {
		index, downstream := i, handler
		next := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			state, ok := req.Context().Value(ingressContinuationKey{}).(*ingressContinuationState)
			if !ok || index >= len(state.called) {
				panic("web: before-routing middleware lost request-local continuation state")
			}
			if state.finished.Load() {
				panic("web: before-routing middleware continued after request completion")
			}
			if state.called[index].Swap(true) {
				panic("web: before-routing middleware called next more than once")
			}
			downstream.ServeHTTP(w, req)
		})
		handler = r.beforeRouting[i](next)
		if handler == nil {
			panic("web: before-routing middleware returned nil handler")
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		state := &ingressContinuationState{called: make([]atomic.Bool, len(r.beforeRouting))}
		defer state.finished.Store(true)
		scoped := req.WithContext(context.WithValue(req.Context(), ingressContinuationKey{}, state))
		handler.ServeHTTP(w, scoped)
	})
}

// AfterRouting validates HTTP actions after Gin path matching. It does
// not create a second Gin-only middleware dispatch path.
func (r *Router) AfterRouting(middlewares ...any) {
	for _, middleware := range middlewares {
		if isNilOrFuncNil(middleware) {
			panic("web: nil after-routing middleware")
		}
		if pre, ok := middleware.(web_types.PreMiddlewareProvider); ok {
			_ = toHandlerProvider(middlewareProvider(pre))
		} else {
			_ = toHandlerProvider(middleware)
		}
	}
	r.afterRouting = append(r.afterRouting, middlewares...)
}

// Group scopes route patterns and middlewares at registration, as in Gitea.
// Gin receives the compiled chain at Methods; it never owns a second scope
// that would reorder group guards ahead of AfterRouting.
func (r *Router) Group(path string, register func(), guards ...any) {
	previousPrefix := r.curGroupPrefix
	previousMiddlewares := r.curMiddlewares
	r.curGroupPrefix += path
	r.curMiddlewares = append(r.curMiddlewares, guards...)
	defer func() {
		r.curGroupPrefix = previousPrefix
		r.curMiddlewares = previousMiddlewares
	}()
	register()
}

func (r *Router) getPattern(path string) string {
	pattern := r.curGroupPrefix + path
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	if pattern == "/" {
		return pattern
	}
	return strings.TrimSuffix(pattern, "/")
}

// Methods compiles the route-scoped middleware chain during registration.
// Routing order: AfterRouting -> group guards -> endpoint actions.
// Gin is still the only route matcher and executor.
func (r *Router) Methods(method, path string, actions ...any) {
	if len(actions) == 0 {
		panic("web: route requires an action")
	}
	// Validate the entire declaration before touching Gin's route tree.
	// Otherwise "GET, " registers GET before failing on its next token.
	candidates := strings.Split(method, ",")
	seen := make(map[string]struct{}, len(candidates))
	for i, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			panic("web: empty HTTP method")
		}
		// Gin accepts only uppercase ASCII letters for HTTP methods.
		// Mirror that precondition before registering any candidate.
		for i := 0; i < len(candidate); i++ {
			if candidate[i] < 'A' || candidate[i] > 'Z' {
				panic(fmt.Sprintf("web: invalid HTTP method %q", candidate))
			}
		}
		if _, duplicate := seen[candidate]; duplicate {
			panic(fmt.Sprintf("web: duplicate HTTP method %s", candidate))
		}
		seen[candidate] = struct{}{}
		candidates[i] = candidate
	}
	// A single registered HTTP provider chain owns all typed actions.
	middlewares, endpoint, _ := wrapMiddlewareAndHandler(r.afterRouting, r.curMiddlewares, actions)
	handlers := []gin.HandlerFunc{ginMatchedHandler(middlewares, endpoint)}
	pattern := r.getPattern(path)
	// Gitea permits comma-separated method registration at the canonical
	// Methods boundary. Compile the handler chain once for every method.
	// As in Chi, lock the BeforeRouting phase before Gin's first
	// registration. Never allow a later pre-matcher middleware to appear
	// effective in source while actually being absent from ingress.
	r.routesRegistered = true
	for _, candidate := range candidates {
		r.group.Handle(candidate, pattern, handlers...)
	}
}

func (r *Router) Get(path string, actions ...any) {
	r.Methods(http.MethodGet, path, actions...)
}
func (r *Router) Head(path string, actions ...any) {
	r.Methods(http.MethodHead, path, actions...)
}
func (r *Router) Post(path string, actions ...any) {
	r.Methods(http.MethodPost, path, actions...)
}
func (r *Router) Put(path string, actions ...any) {
	r.Methods(http.MethodPut, path, actions...)
}
func (r *Router) Patch(path string, actions ...any) {
	r.Methods(http.MethodPatch, path, actions...)
}
func (r *Router) Delete(path string, actions ...any) {
	r.Methods(http.MethodDelete, path, actions...)
}
