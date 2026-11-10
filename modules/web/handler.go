package web

import (
	"fmt"
	"net/http"

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
