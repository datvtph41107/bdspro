package web

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"github.com/gin-gonic/gin"
)

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
		r.afterRouting = append(r.afterRouting, r.toHandlerProvider(middleware))
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
		handlers = append(handlers, r.handlers(r.curMiddlewares)...)
	}
	handlers = append(handlers, r.handlers(actions)...)
	r.group.Handle(method, r.getPattern(path), handlers...)
}

func (r *Router[T]) Get(path string, actions ...func(T)) {
	r.Methods(http.MethodGet, path, actions...)
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
