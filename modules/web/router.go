package web

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/gin-gonic/gin"
)

type routeDef struct {
	method      string
	path        string
	middlewares []middlewareProvider
	endpoint    http.HandlerFunc
}

type mountDef struct {
	prefix      string
	router      *Router
	middlewares []middlewareProvider
}

// Router owns route declarations; Gin exclusively owns route matching.
type Router struct {
	engine         *gin.Engine
	afterRouting   []any
	curGroupPrefix string
	curMiddlewares []any
	routes         []routeDef
	mounts         []mountDef
	compileOnce    sync.Once
	compiled       bool
}

func NewRouter() *Router {
	engine := gin.New()
	engine.Use(gin.Recovery())
	return &Router{engine: engine}
}

func (r *Router) ensureMutable() {
	if r.compiled {
		panic("web: route registration after compilation")
	}
}

func (r *Router) AfterRouting(middlewares ...any) {
	r.ensureMutable()
	r.afterRouting = append(r.afterRouting, middlewares...)
}

func (r *Router) Group(pattern string, fn func(), middlewares ...any) {
	r.ensureMutable()
	if fn == nil {
		panic("web: nil group callback")
	}
	previousPrefix, previousMiddlewares := r.curGroupPrefix, r.curMiddlewares
	r.curGroupPrefix += pattern
	r.curMiddlewares = append(append([]any(nil), r.curMiddlewares...), middlewares...)
	defer func() {
		r.curGroupPrefix = previousPrefix
		r.curMiddlewares = previousMiddlewares
	}()
	fn()
}

func (r *Router) getPattern(pattern string) string {
	path := r.curGroupPrefix + pattern
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path == "/" {
		return path
	}
	return strings.TrimSuffix(path, "/")
}

// ginPattern supports literal paths and simple Chi-style named parameters.
// More advanced Chi route patterns are explicitly rejected, not weakened.
func ginPattern(pattern string) string {
	var path strings.Builder
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '{':
			if i == 0 || pattern[i-1] != '/' {
				panic(fmt.Sprintf("web: route parameter must begin a segment: %q", pattern))
			}
			end := strings.IndexByte(pattern[i:], '}')
			if end < 0 {
				panic(fmt.Sprintf("web: unclosed route parameter: %q", pattern))
			}
			name := pattern[i+1 : i+end]
			next := i + end + 1
			if name == "" || strings.ContainsAny(name, ":/{}") || (next < len(pattern) && pattern[next] != '/') {
				panic(fmt.Sprintf("web: unsupported Chi route pattern %q", pattern))
			}
			path.WriteByte(':')
			path.WriteString(name)
			i = next
		case '}':
			panic(fmt.Sprintf("web: unmatched route brace: %q", pattern))
		default:
			path.WriteByte(pattern[i])
			i++
		}
	}
	return path.String()
}

func (r *Router) Methods(methods, pattern string, h ...any) {
	r.ensureMutable()
	middlewares, endpoint := wrapMiddlewareAndHandler(r.afterRouting, r.curMiddlewares, h)
	fullPattern := r.getPattern(pattern)
	for _, method := range strings.Split(methods, ",") {
		method = strings.TrimSpace(method)
		if method == "" {
			panic("web: empty HTTP method")
		}
		r.routes = append(r.routes, routeDef{
			method:      method,
			path:        fullPattern,
			middlewares: middlewares,
			endpoint:    endpoint,
		})
	}
}

func (r *Router) Get(path string, h ...any)    { r.Methods(http.MethodGet, path, h...) }
func (r *Router) Post(path string, h ...any)   { r.Methods(http.MethodPost, path, h...) }
func (r *Router) Put(path string, h ...any)    { r.Methods(http.MethodPut, path, h...) }
func (r *Router) Patch(path string, h ...any)  { r.Methods(http.MethodPatch, path, h...) }
func (r *Router) Delete(path string, h ...any) { r.Methods(http.MethodDelete, path, h...) }
func (r *Router) Head(path string, h ...any)   { r.Methods(http.MethodHead, path, h...) }

// Mount captures middleware from the current parent declaration scope.
// Routes are flattened once at startup; there is no second route matcher.
func (r *Router) Mount(pattern string, sub *Router) {
	r.ensureMutable()
	if sub == nil || sub == r {
		panic("web: invalid subrouter")
	}
	providers := make([]middlewareProvider, 0, len(r.afterRouting)+len(r.curMiddlewares))
	providers = appendPre(providers, r.afterRouting)
	providers = appendPre(providers, r.curMiddlewares)
	providers = appendNormal(providers, r.afterRouting)
	providers = appendNormal(providers, r.curMiddlewares)
	r.mounts = append(r.mounts, mountDef{
		prefix:      r.getPattern(pattern),
		router:      sub,
		middlewares: providers,
	})
}

func joinRoute(prefix, path string) string {
	if prefix == "" || prefix == "/" {
		return path
	}
	if path == "/" {
		return prefix
	}
	return strings.TrimSuffix(prefix, "/") + path
}

func (r *Router) installInto(engine *gin.Engine, prefix string, parent []middlewareProvider, seen map[*Router]bool) {
	if seen[r] {
		panic("web: cyclic router mount")
	}
	seen[r] = true
	defer delete(seen, r)
	r.compiled = true
	for _, route := range r.routes {
		full := ginPattern(joinRoute(prefix, route.path))
		chain := append(append([]middlewareProvider(nil), parent...), route.middlewares...)
		handler := composeHandler(chain, route.endpoint)
		engine.Handle(route.method, full, func(c *gin.Context) {
			for _, param := range c.Params {
				c.Request.SetPathValue(param.Key, param.Value)
			}
			handler.ServeHTTP(c.Writer, c.Request)
		})
	}
	for _, mount := range r.mounts {
		inherited := append(append([]middlewareProvider(nil), parent...), mount.middlewares...)
		mount.router.installInto(engine, joinRoute(prefix, mount.prefix), inherited, seen)
	}
}

// Compile validates and installs all declarations before accepting traffic.
// All route registration must finish before this method is called.
func (r *Router) Compile() {
	r.compileOnce.Do(func() {
		r.installInto(r.engine, "", nil, map[*Router]bool{})
	})
}

func (r *Router) SetTrustedProxies(proxies []string) error {
	return r.engine.SetTrustedProxies(proxies)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Compile()
	if reqctx.FromContext(req.Context()) != nil {
		r.engine.ServeHTTP(w, req)
		return
	}
	scoped, finished := reqctx.NewRequestContext(req.Context())
	defer finished()
	r.engine.ServeHTTP(w, req.WithContext(scoped))
}
