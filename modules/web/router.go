package web

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/datvtph41107/bdspro/modules/reqctx"
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

// FormStore owns request-local data through reqctx; web does not manage a
// separate registry or a global cache. The form is exposed only after binding.
type FormStore = reqctx.ContextDataProvider

const formKey = "__form"

type JSONFormContext interface {
	FormStore
	BindJSON(value any) error
}

// BindJSON is the single owner of typed JSON decoding and form publication.
// The API boundary remains responsible for reporting validation failures.
func BindJSON[T any](ctx JSONFormContext) (*T, error) {
	form := new(T)
	if err := ctx.BindJSON(form); err != nil {
		return form, err
	}
	SetForm(ctx, form)
	return form, nil
}

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
