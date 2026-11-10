package web

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// Gin compatibility is intentionally isolated from Gitea-style HTTP
// handler compilation in handler.go and route registration in router.go.
// It is the sole transport bridge until callers migrate to the final
// non-generic HTTP Router contract.

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

// ginScopedHandler is the one matched-route Gin/net-http execution boundary.
// Each middleware can pass a new *http.Request and ResponseWriter to next.
// Bind those values for the duration of the downstream handler so Gin-backed
// APIContext accessors see the current request and do not bypass HTTP wrappers.
// Restore the enclosing scope on return, including on early termination.
func ginScopedHandler(c *gin.Context, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		previousRequest, previousWriter := c.Request, c.Writer
		c.Request = req
		if w != previousWriter {
			c.Writer = newGinDelegatingWriter(w, previousWriter)
		}
		defer func() {
			c.Request = previousRequest
			c.Writer = previousWriter
		}()
		handler.ServeHTTP(w, req)
	})
}

// matchedRouteScope binds one Gin matcher invocation to the HTTP middleware
// chain compiled at registration. It is request-local, never process-global.
// Gin is still the only route matcher and APIContext remains request-owned.
type matchedRouteKey struct{}

type matchedRouteScope struct {
	gin      *gin.Context
	incoming []*http.Request
	called   []atomic.Bool
	finished atomic.Bool
}

// GinContextFromRequest exposes the matched transport context to the single
// services/context bridge. It must only be used after Gin resolves the route;
// domain handlers consume *context.APIContext, not *gin.Context.
func GinContextFromRequest(req *http.Request) *gin.Context {
	scope, ok := req.Context().Value(matchedRouteKey{}).(*matchedRouteScope)
	if !ok || scope == nil {
		return nil
	}
	return scope.gin
}

func matchedRouteFromRequest(req *http.Request) *matchedRouteScope {
	scope, ok := req.Context().Value(matchedRouteKey{}).(*matchedRouteScope)
	if !ok || scope == nil {
		panic("web: matched route middleware lost request-local Gin scope")
	}
	return scope
}

// ginMatchedHandler compiles the Gitea-style HTTP middleware providers ONCE,
// when Methods registers a route. Only the Gin transport binding and the
// one-shot continuation flags are created for each request.
//
// Each provider may forward a new *http.Request or ResponseWriter. An
// existing GinActions handler can instead replace c.Request directly (after
// live-session verification); that replacement takes precedence, matching
// the prior bridge's propagation rule.
func ginMatchedHandler(providers []middlewareProvider, endpoint http.HandlerFunc) gin.HandlerFunc {
	// Bind the Gin execution boundary without rebuilding providers per
	// request. The canonical HTTP compiler below composes this list once.
	bound := make([]middlewareProvider, len(providers))
	for i, provider := range providers {
		index, middleware := i, provider
		bound[i] = func(next http.Handler) http.Handler {
			forward := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				scope := matchedRouteFromRequest(req)
				if scope.finished.Load() {
					panic("web: matched route middleware continued after request completion")
				}
				if scope.called[index].Swap(true) {
					panic("web: HTTP middleware called next more than once")
				}
				// Gin protocol actions may update c.Request. Ordinary HTTP
				// middleware instead propagates a new request to next.
				if scope.gin.Request != scope.incoming[index] {
					req = scope.gin.Request
				}
				ginScopedHandler(scope.gin, next).ServeHTTP(w, req)
			})
			compiled := middleware(forward)
			if compiled == nil {
				panic("web: matched route middleware returned nil handler")
			}
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				scope := matchedRouteFromRequest(req)
				if scope.finished.Load() {
					panic("web: matched route middleware executed after request completion")
				}
				scope.incoming[index] = req
				ginScopedHandler(scope.gin, compiled).ServeHTTP(w, req)
			})
		}
	}
	endpointHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		scope := matchedRouteFromRequest(req)
		ginScopedHandler(scope.gin, endpoint).ServeHTTP(w, req)
	})
	chain := compileMiddlewaresHandler(bound, endpointHandler)
	return func(c *gin.Context) {
		scope := &matchedRouteScope{
			gin:      c,
			incoming: make([]*http.Request, len(providers)),
			called:   make([]atomic.Bool, len(providers)),
		}
		defer scope.finished.Store(true)
		request := c.Request.WithContext(context.WithValue(c.Request.Context(), matchedRouteKey{}, scope))
		c.Request = request
		chain.ServeHTTP(c.Writer, request)
	}
}
