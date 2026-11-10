package web

import (
	"fmt"
	"net/http"
	"reflect"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"github.com/gin-gonic/gin"
)

// responseWriter observes whether an HTTP action committed a response.
// A selected Gin status is not enough: only an explicit HTTP write commits.
// Unwrap permits http.ResponseController to reach the underlying writer.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WrittenStatus() int { return w.status }

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// middlewareProvider is the net/http continuation contract used by Gitea.
// It is independent of both the Gin route matcher and typed API contexts.
type middlewareProvider = func(next http.Handler) http.Handler

// toHandlerProvider validates native HTTP middleware once at registration.
// A typed API action is handled by the existing route compiler until the
// APIContext provider migration is completed.
func toHandlerProvider(handler any) middlewareProvider {
	switch h := handler.(type) {
	case func(http.Handler) http.Handler:
		if h == nil {
			panic("web: nil HTTP middleware")
		}
		return middlewareProvider(h)
	case func(http.Handler) http.HandlerFunc:
		if h == nil {
			panic("web: nil HTTP middleware")
		}
		return func(next http.Handler) http.Handler { return h(next) }
	case func(http.ResponseWriter, *http.Request):
		if h == nil {
			panic("web: nil HTTP action")
		}
		return toHandlerProvider(http.HandlerFunc(h))
	case http.HandlerFunc:
		if h == nil {
			panic("web: nil HTTP action")
		}
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				tracked, ok := w.(*responseWriter)
				if !ok {
					tracked = &responseWriter{ResponseWriter: w}
				}
				h(tracked, req)
				if next != nil && tracked.WrittenStatus() == 0 {
					next.ServeHTTP(tracked, req)
				}
			})
		}
	default:
		panic(fmt.Sprintf("web: unsupported HTTP middleware %T", handler))
	}
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
