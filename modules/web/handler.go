package web

import "github.com/gin-gonic/gin"

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
			fn(r.getContext(c))
			// Gitea stops the typed handler chain after a committed response.
			// Gin JSON/Write does not itself abort following route handlers.
			// Selecting Status without writing headers must not abort.
			if c.Writer.Written() && !c.IsAborted() {
				c.Abort()
			}
		})
	}
	return handlers
}

func (r *Router[T]) handlersIfAny(actions []func(T)) []gin.HandlerFunc {
	if len(actions) == 0 {
		return nil
	}
	return r.handlers(actions)
}
