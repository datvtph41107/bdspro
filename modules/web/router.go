package web

import (
	"fmt"
	"net/http"
	"reflect"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"github.com/gin-gonic/gin"
)

// Router maps typed HTTP actions into Gin at registration time. Gin remains
// the only middleware executor. Do not mutate this router after startup.
type Router[T any] struct {
	group      *gin.RouterGroup
	getContext func(*gin.Context) T
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

// Group follows Gitea's Group(path, func() {...}, guards...) declaration.
// Its mutable group state exists only during single-threaded route registration.
func (r *Router[T]) Group(path string, register func(), guards ...func(T)) {
	parent := r.group
	r.group = parent.Group(path, r.handlersIfAny(guards)...)
	defer func() { r.group = parent }()
	register()
}

// Methods is the one HTTP method registration boundary. Gin owns route
// matching and execution; this facade owns conversion of typed API actions.
func (r *Router[T]) Methods(method, path string, actions ...func(T)) {
	r.group.Handle(method, path, r.handlers(actions)...)
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
