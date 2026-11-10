package web

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"github.com/gin-gonic/gin"
)

// ginDelegatingWriter adapts a response writer supplied by net/http
// middleware to Gin's response writer contract. It ensures downstream Gin
// actions write through HTTP middleware wrappers rather than bypassing them.
type ginDelegatingWriter struct {
	http.ResponseWriter
	original gin.ResponseWriter
	status   int
	size     int
}

var _ gin.ResponseWriter = (*ginDelegatingWriter)(nil)

func newGinDelegatingWriter(w http.ResponseWriter, original gin.ResponseWriter) *ginDelegatingWriter {
	return &ginDelegatingWriter{
		ResponseWriter: w,
		original:       original,
		status:         original.Status(),
		size:           original.Size(),
	}
}

func (w *ginDelegatingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *ginDelegatingWriter) WriteHeader(status int) {
	if status > 0 && !w.Written() {
		w.status = status
		// Gin defers header commitment. Keep the underlying writer's
		// selected status synchronized for status-only HTTP actions.
		w.original.WriteHeader(status)
	}
}

func (w *ginDelegatingWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *ginDelegatingWriter) Write(body []byte) (int, error) {
	w.WriteHeaderNow()
	n, err := w.ResponseWriter.Write(body)
	w.size += n
	return n, err
}

func (w *ginDelegatingWriter) WriteString(body string) (int, error) {
	w.WriteHeaderNow()
	n, err := io.WriteString(w.ResponseWriter, body)
	w.size += n
	return n, err
}

func (w *ginDelegatingWriter) Status() int   { return w.status }
func (w *ginDelegatingWriter) Size() int     { return w.size }
func (w *ginDelegatingWriter) Written() bool { return w.size >= 0 }

func (w *ginDelegatingWriter) Flush() {
	w.WriteHeaderNow()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *ginDelegatingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (w *ginDelegatingWriter) CloseNotify() <-chan bool {
	if notifier, ok := w.ResponseWriter.(http.CloseNotifier); ok {
		return notifier.CloseNotify()
	}
	return w.original.CloseNotify()
}

func (w *ginDelegatingWriter) Pusher() http.Pusher {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}

// ginHandlerProvider is the Gin execution boundary for a net/http provider.
// It preserves both the Request and any ResponseWriter supplied to next.
func ginHandlerProvider(provider middlewareProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		continued := false
		next := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if continued {
				panic("web: HTTP middleware called next more than once")
			}
			continued = true
			c.Request = req
			original := c.Writer
			// Gitea's status tracker does not need to become an extra Gin
			// writer when no response has been committed and no custom
			// ResponseWriter was supplied by an HTTP middleware.
			if tracked, ok := w.(*responseWriter); ok &&
				tracked.WrittenStatus() == 0 && tracked.ResponseWriter == original {
				w = original
			}
			if w != original {
				c.Writer = newGinDelegatingWriter(w, original)
				defer func() { c.Writer = original }()
			}
			c.Next()
		})
		provider(next).ServeHTTP(c.Writer, c.Request)
		if !continued {
			c.Abort()
		}
	}
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

// Router maps typed HTTP actions into Gin at registration time. Gin remains
// the only middleware executor. Do not mutate this router after startup.
type Router[T any] struct {
	group          *gin.RouterGroup
	getContext     func(*gin.Context) T
	afterRouting   []gin.HandlerFunc
	curGroupPrefix string
	curMiddlewares []func(T)
}

func NewRouter[T any](group *gin.RouterGroup, getContext func(*gin.Context) T) *Router[T] {
	if group == nil || getContext == nil {
		panic("web: route group and context accessor are required")
	}
	// Gitea prechecks that typed actions have a ResponseStatusProvider.
	// Gin-only compatibility actions use Gin's own response writer instead;
	// do not allocate a reflection registry for this already typed router.
	typ := reflect.TypeFor[T]()
	if typ != reflect.TypeFor[*gin.Context]() && !typ.Implements(reflect.TypeFor[web_types.ResponseStatusProvider]()) {
		panic(fmt.Sprintf("web: action context %s must implement ResponseStatusProvider", typ))
	}
	return &Router[T]{group: group, getContext: getContext}
}

// AfterRouting installs middleware after Gin has resolved the route, before
// group guards, binding, and the endpoint action. A request context belongs to
// the selected API route rather than to unmatched or non-API requests.
// Supports native Gin middleware at the framework boundary and typed actions;
// unsupported signatures fail during route registration.
// Register these middlewares before the routes that should receive them.
func (r *Router[T]) AfterRouting(middlewares ...any) {
	for _, middleware := range middlewares {
		switch middleware.(type) {
		case func(http.Handler) http.Handler, func(http.Handler) http.HandlerFunc, http.HandlerFunc, func(http.ResponseWriter, *http.Request):
			r.afterRouting = append(r.afterRouting, ginHandlerProvider(toHandlerProvider(middleware)))
		default:
			if isRegisteredTypedHandler(middleware) {
				r.afterRouting = append(r.afterRouting, ginHandlerProvider(toHandlerProvider(middleware)))
			} else {
				r.afterRouting = append(r.afterRouting, r.toGinHandler(middleware))
			}
		}
	}
}

// Group scopes route patterns and middlewares at registration, as in Gitea.
// Gin receives the compiled chain at Methods; it never owns a second scope
// that would reorder group guards ahead of AfterRouting.
func (r *Router[T]) Group(path string, register func(), guards ...func(T)) {
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

func (r *Router[T]) getPattern(path string) string {
	pattern := r.curGroupPrefix + path
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	if pattern == "/" {
		return pattern
	}
	return strings.TrimSuffix(pattern, "/")
}

// compileActions selects the registered net/http typed handler contract
// when available. Legacy Gin-only test contexts use the compatibility path;
// normal APIContext routes use the package-level provider from handler.go.
func (r *Router[T]) compileActions(actions []func(T)) []gin.HandlerFunc {
	if _, registered := responseStatusProviders[reflect.TypeFor[T]()]; !registered {
		return r.handlers(actions)
	}
	out := make([]gin.HandlerFunc, 0, len(actions))
	for _, action := range actions {
		provider := ginHandlerProvider(toHandlerProvider(action))
		out = append(out, func(c *gin.Context) {
			// The old Router contract explicitly supplied getContext. Keep
			// that initialization boundary for routes registered without
			// APIContexter (including direct route callers) until P02
			// makes APIContexter mandatory for the non-generic Router.
			r.getContext(c)
			provider(c)
		})
	}
	return out
}

// Methods compiles the route-scoped middleware chain during registration.
// Routing order: AfterRouting -> group guards -> endpoint actions.
// Gin is still the only route matcher and executor.
func (r *Router[T]) Methods(method, path string, actions ...func(T)) {
	if len(actions) == 0 {
		panic("web: route requires an action")
	}
	handlers := make([]gin.HandlerFunc, 0, len(r.afterRouting)+len(r.curMiddlewares)+len(actions))
	handlers = append(handlers, r.afterRouting...)
	if len(r.curMiddlewares) > 0 {
		handlers = append(handlers, r.compileActions(r.curMiddlewares)...)
	}
	handlers = append(handlers, r.compileActions(actions)...)
	pattern := r.getPattern(path)
	// Gitea permits comma-separated method registration at the canonical
	// Methods boundary. Compile the handler chain once for every method.
	for _, candidate := range strings.Split(method, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			panic("web: empty HTTP method")
		}
		r.group.Handle(candidate, pattern, handlers...)
	}
}

func (r *Router[T]) Get(path string, actions ...func(T)) {
	r.Methods(http.MethodGet, path, actions...)
}
func (r *Router[T]) Head(path string, actions ...func(T)) {
	r.Methods(http.MethodHead, path, actions...)
}
func (r *Router[T]) Post(path string, actions ...func(T)) {
	r.Methods(http.MethodPost, path, actions...)
}
func (r *Router[T]) Put(path string, actions ...func(T)) {
	r.Methods(http.MethodPut, path, actions...)
}
func (r *Router[T]) Patch(path string, actions ...func(T)) {
	r.Methods(http.MethodPatch, path, actions...)
}
func (r *Router[T]) Delete(path string, actions ...func(T)) {
	r.Methods(http.MethodDelete, path, actions...)
}

// toGinHandler validates Gin-compatible handler kinds at registration and
// converts them to the single Gin execution chain. The router owns
// declarations; this file owns middleware/action adaptation, as in Gitea.
func (r *Router[T]) toGinHandler(middleware any) gin.HandlerFunc {
	switch handler := middleware.(type) {
	case gin.HandlerFunc:
		if handler == nil {
			panic("web: nil after-routing middleware")
		}
		return handler
	case func(*gin.Context):
		if handler == nil {
			panic("web: nil after-routing middleware")
		}
		// A plain function is a typed action when T is *gin.Context.
		// Treat it as such so a committed response terminates the chain.
		if reflect.TypeFor[T]() == reflect.TypeFor[*gin.Context]() {
			return r.handlers([]func(T){any(handler).(func(T))})[0]
		}
		return gin.HandlerFunc(handler)
	case func(T):
		if handler == nil {
			panic("web: nil after-routing middleware")
		}
		return r.handlers([]func(T){handler})[0]
	default:
		panic(fmt.Sprintf("web: unsupported after-routing middleware %T", middleware))
	}
}

// handlers adapts Gitea-style typed route actions into the single Gin
// execution chain. This package owns the conversion and continuation policy,
// while Gin remains the only HTTP middleware and response executor.
func (r *Router[T]) handlers(actions []func(T)) []gin.HandlerFunc {
	if len(actions) == 0 {
		panic("web: route requires an action")
	}
	handlers := make([]gin.HandlerFunc, 0, len(actions))
	for _, action := range actions {
		if action == nil {
			panic("web: nil route action")
		}
		fn := action
		handlers = append(handlers, func(c *gin.Context) {
			ctx := r.getContext(c)
			fn(ctx)
			// The typed API context owns the response lifecycle, as in Gitea's
			// hasResponseBeenWritten. This checks the committed response, not
			// Gin's selected-but-unwritten status code. Only legacy *gin.Context
			// actions fall back to Gin's writer.
			if c.IsAborted() {
				return
			}
			written := false
			if status, ok := any(ctx).(web_types.ResponseStatusProvider); ok {
				written = status.WrittenStatus() != 0
			} else {
				written = c.Writer.Written()
			}
			if written {
				c.Abort()
			}
		})
	}
	return handlers
}
