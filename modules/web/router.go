package web

import "github.com/gin-gonic/gin"

// Router maps typed HTTP actions into Gin at registration time. Gin remains
// the only middleware executor. Do not mutate this router after startup.
type Router[T any] struct {
	group      *gin.RouterGroup
	getContext func(*gin.Context) T
}

func NewRouter[T any](group *gin.RouterGroup, getContext func(*gin.Context) T) *Router[T] {
	return &Router[T]{group: group, getContext: getContext}
}

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
		handlers = append(handlers, func(c *gin.Context) { fn(r.getContext(c)) })
	}
	return handlers
}

// Group follows Gitea's Group(path, func() {...}, guards...) declaration.
// Its mutable group state exists only during single-threaded route registration.
func (r *Router[T]) Group(path string, register func(), guards ...func(T)) {
	parent := r.group
	r.group = parent.Group(path, r.handlersIfAny(guards)...)
	defer func() { r.group = parent }()
	register()
}

func (r *Router[T]) handlersIfAny(actions []func(T)) []gin.HandlerFunc {
	if len(actions) == 0 {
		return nil
	}
	return r.handlers(actions)
}

func (r *Router[T]) Get(path string, actions ...func(T))  { r.group.GET(path, r.handlers(actions)...) }
func (r *Router[T]) Post(path string, actions ...func(T)) { r.group.POST(path, r.handlers(actions)...) }
func (r *Router[T]) Put(path string, actions ...func(T))  { r.group.PUT(path, r.handlers(actions)...) }
func (r *Router[T]) Patch(path string, actions ...func(T)) {
	r.group.PATCH(path, r.handlers(actions)...)
}
func (r *Router[T]) Delete(path string, actions ...func(T)) {
	r.group.DELETE(path, r.handlers(actions)...)
}
