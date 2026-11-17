package web

import (
	"fmt"
	"net/http"
	"net/netip"
	"reflect"
	"strings"
	"sync"


	"github.com/datvtph41107/bdspro/modules/reqctx"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"github.com/go-chi/chi/v5"
)

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

// Router owns the single Chi matcher and compiled HTTP action chains.
type Router struct {
	chiRouter         *chi.Mux
	chiTrustedProxies []netip.Prefix
	ingressBound      bool
	routesRegistered  bool
	mounted           bool
	afterRouting      []any
	curGroupPrefix    string
	curMiddlewares    []any
	serveOnce         sync.Once
	serveHandler      http.Handler
}

func NewRouter() *Router { return &Router{chiRouter: chi.NewRouter()} }

func (r *Router) SetTrustedProxies(proxies []string) error {
	if r.ingressBound {
		panic("web: trusted proxies must be configured before serving")
	}
	return r.setChiTrustedProxies(proxies)
}

// Mount structurally mounts a Chi subrouter. Parent middleware executes
// before child middleware; a mounted child cannot be mutated afterward.
func (r *Router) Mount(pattern string, sub *Router) {
	if r.mounted || r.ingressBound {
		panic("web: cannot mount after child freeze or ingress composition")
	}
	if sub == nil || sub == r || sub.chiRouter == r.chiRouter {
		panic("web: invalid or self-mounted router")
	}
	if sub.ingressBound || !sub.routesRegistered {
		panic("web: child must have routes and must not be serving")
	}
	prefix := r.getPattern(pattern)
	parentMiddleware := make([]middlewareProvider, 0, len(r.afterRouting)+len(r.curMiddlewares))
	parentMiddleware = wrapMiddlewareAppendPre(parentMiddleware, r.afterRouting)
	parentMiddleware = wrapMiddlewareAppendPre(parentMiddleware, r.curMiddlewares)
	parentMiddleware = wrapMiddlewareAppendNormal(parentMiddleware, r.afterRouting)
	parentMiddleware = wrapMiddlewareAppendNormal(parentMiddleware, r.curMiddlewares)
	r.chiRouter.With(parentMiddleware...).Mount(prefix, sub.chiRouter)
	r.routesRegistered = true
	sub.mounted = true
}

// ServeHTTP invokes the same HTTP pipeline exposed through WrapIngress.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.serveOnce.Do(func() {
		r.serveHandler = r.WrapIngress()
	})
	r.serveHandler.ServeHTTP(w, req)
}

// BeforeRouting installs HTTP middleware before matching, including 404/405.
func (r *Router) BeforeRouting(middlewares ...any) {
	if r.mounted || r.ingressBound || r.routesRegistered {
		panic("web: BeforeRouting must be declared before routes and serving")
	}
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
			panic(fmt.Sprintf("web: BeforeRouting requires HTTP middleware, got %T", middleware))
		}
	}
	r.chiRouter.Use(compiled...)
}

func (r *Router) WrapIngress() http.Handler {
	if r.ingressBound {
		panic("web: ingress already composed")
	}
	r.ingressBound = true
	return r.wrapChiIngress()
}

// AfterRouting declares HTTP actions applied after route matching.
func (r *Router) AfterRouting(middlewares ...any) {
	if r.mounted {
		panic("web: cannot change AfterRouting after mounting child router")
	}
	// Route handlers are compiled by Methods at registration time. A late
	// AfterRouting declaration would silently affect only subsequent routes.
	if r.routesRegistered {
		panic("web: AfterRouting middleware must be declared before routes")
	}
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

// Group scopes route patterns and middleware at registration, as in Gitea.
func (r *Router) Group(path string, register func(), guards ...any) {
	if r.mounted {
		panic("web: cannot register Group after mounting child router")
	}
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

// Methods compiles a Gitea-style typed HTTP handler chain into Chi routes.
func (r *Router) Methods(method, path string, actions ...any) {
	if r.mounted || r.ingressBound {
		panic("web: cannot register methods after mount or serving")
	}
	if len(actions) == 0 {
		panic("web: route requires an action")
	}
	candidates := strings.Split(method, ",")
	seen := make(map[string]struct{}, len(candidates))
	for i, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			panic("web: empty HTTP method")
		}
		for j := 0; j < len(candidate); j++ {
			if candidate[j] < 'A' || candidate[j] > 'Z' {
				panic(fmt.Sprintf("web: invalid HTTP method %q", candidate))
			}
		}
		if _, duplicate := seen[candidate]; duplicate {
			panic(fmt.Sprintf("web: duplicate HTTP method %s", candidate))
		}
		seen[candidate] = struct{}{}
		candidates[i] = candidate
	}
	middlewares, endpoint, _ := wrapMiddlewareAndHandler(r.afterRouting, r.curMiddlewares, actions)
	pattern := r.getPattern(path)
	r.routesRegistered = true
	for _, candidate := range candidates {
		r.chiRouter.With(middlewares...).Method(candidate, pattern, endpoint)
	}
}

// Combo declares multiple HTTP methods at one path with shared actions,
// matching Gitea's route-level composition syntax.
func (r *Router) Combo(pattern string, handlers ...any) *Combo {
	if r.mounted {
		panic("web: cannot register Combo after mounting child router")
	}
	return &Combo{r: r, pattern: pattern, h: handlers}
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
