package web

import (
	"fmt"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"github.com/gin-gonic/gin"
	"reflect"
)

// toHandlerProvider validates handler kinds once at route registration and
// converts them to the single Gin execution chain. The router owns
// declarations; this file owns middleware/action adaptation, as in Gitea.
func (r *Router[T]) toHandlerProvider(middleware any) gin.HandlerFunc {
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
