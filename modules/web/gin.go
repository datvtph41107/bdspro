package web

import (
	"bufio"
	"io"
	"net"
	"net/http"

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
