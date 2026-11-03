package web

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Router mirrors the organization and public route API of Gitea's modules/web.
// Chi performs method/path matching; Gitea-style typed adaptation is compiled
// when a route is declared, rather than reconstructing middleware per request.
type Router struct {
	chiRouter      *chi.Mux
	afterRouting   []any
	curGroupPrefix string
	curMiddlewares []any
}

func NewRouter() *Router { return &Router{chiRouter: chi.NewRouter()} }

// BeforeRouting is for protocol-level middleware that runs before route matching.
// As with chi.Use, call before registering routes.
func (r *Router) BeforeRouting(middlewares ...any) {
	for _, m := range middlewares {
		if m != nil {
			r.chiRouter.Use(toHandlerProvider(m))
		}
	}
}

// AfterRouting runs only once a route has matched and path parameters are set.
func (r *Router) AfterRouting(middlewares ...any) {
	r.afterRouting = append(r.afterRouting, middlewares...)
}

func (r *Router) Group(pattern string, fn func(), middlewares ...any) {
	if fn == nil {
		panic("web: nil group callback")
	}
	oldPrefix, oldMiddlewares := r.curGroupPrefix, r.curMiddlewares
	r.curGroupPrefix = oldPrefix + pattern
	r.curMiddlewares = append(append([]any(nil), oldMiddlewares...), middlewares...)
	defer func() { r.curGroupPrefix, r.curMiddlewares = oldPrefix, oldMiddlewares }()
	fn()
}

func (r *Router) pattern(path string) string {
	joined := r.curGroupPrefix + path
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	if joined == "/" {
		return joined
	}
	return strings.TrimSuffix(joined, "/")
}

func (r *Router) Methods(methods, path string, handlers ...any) {
	if len(handlers) == 0 {
		panic("web: missing endpoint")
	}
	endpoint := adaptHandlers(r.curMiddlewares, r.afterRouting, handlers)
	for _, method := range strings.Split(methods, ",") {
		method = strings.TrimSpace(method)
		if method == "" {
			panic("web: empty HTTP method")
		}
		r.chiRouter.Method(method, r.pattern(path), endpoint)
	}
}

func (r *Router) Mount(pattern string, sub *Router) {
	if sub == nil {
		panic("web: nil router")
	}
	providers := make([]middlewareProvider, 0, len(r.curMiddlewares)+len(r.afterRouting))
	for _, middleware := range append(append([]any(nil), r.afterRouting...), r.curMiddlewares...) {
		if middleware != nil {
			providers = append(providers, toHandlerProvider(middleware))
		}
	}
	r.chiRouter.With(providers...).Mount(r.pattern(pattern), sub.chiRouter)
}
func (r *Router) Get(path string, h ...any)     { r.Methods(http.MethodGet, path, h...) }
func (r *Router) Post(path string, h ...any)    { r.Methods(http.MethodPost, path, h...) }
func (r *Router) Put(path string, h ...any)     { r.Methods(http.MethodPut, path, h...) }
func (r *Router) Patch(path string, h ...any)   { r.Methods(http.MethodPatch, path, h...) }
func (r *Router) Delete(path string, h ...any)  { r.Methods(http.MethodDelete, path, h...) }
func (r *Router) Head(path string, h ...any)    { r.Methods(http.MethodHead, path, h...) }
func (r *Router) Options(path string, h ...any) { r.Methods(http.MethodOptions, path, h...) }
func (r *Router) Combo(path string, h ...any) *Combo {
	return &Combo{r: r, path: path, base: append([]any(nil), h...)}
}
func (r *Router) NotFound(h http.HandlerFunc)                        { r.chiRouter.NotFound(h) }
func (r *Router) MethodNotAllowed(h http.HandlerFunc)                { r.chiRouter.MethodNotAllowed(h) }
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) { r.chiRouter.ServeHTTP(w, req) }
