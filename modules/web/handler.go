package web

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync"

	"github.com/datvtph41107/bdspro/modules/web/types"
)

// middlewareProvider is exactly the net/http middleware contract.
type middlewareProvider = func(http.Handler) http.Handler

var (
	providersMu        sync.RWMutex
	providers          = map[reflect.Type]func(*http.Request) types.ResponseStatusProvider{}
	typedHandlers      = map[reflect.Type]func(any) middlewareProvider{}
	httpRequestType    = reflect.TypeFor[*http.Request]()
	responseWriterType = reflect.TypeFor[http.ResponseWriter]()
)

// RegisterResponseStatusProvider registers a typed handler context resolver.
// Call during initialization, before the router is registered or used.
// A registry isolates the generic routing machinery from services/context.
func RegisterResponseStatusProvider[T types.ResponseStatusProvider](resolve func(*http.Request) T) {
	if resolve == nil {
		panic("web: nil response status provider")
	}
	key := reflect.TypeFor[T]()
	providersMu.Lock()
	defer providersMu.Unlock()
	if _, exists := providers[key]; exists {
		panic("web: duplicate status provider: " + key.String())
	}
	providers[key] = func(r *http.Request) types.ResponseStatusProvider { return resolve(r) }
	// Compile the common func(T) dispatch once. Typed APIContext handlers
	// use no reflect.Call or dynamic argument allocations on the hot path.
	typedHandlers[reflect.TypeFor[func(T)]()] = func(handler any) middlewareProvider {
		fn := handler.(func(T))
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := resolve(r)
				fn(ctx)
				if next != nil && ctx.WrittenStatus() == 0 {
					next.ServeHTTP(w, r)
				}
			})
		}
	}
}

func providerFor(t reflect.Type) (func(*http.Request) types.ResponseStatusProvider, bool) {
	providersMu.RLock()
	defer providersMu.RUnlock()
	resolve, ok := providers[t]
	return resolve, ok
}

func toHandlerProvider(h any) middlewareProvider {
	if h == nil {
		panic("web: nil handler")
	}
	switch fn := h.(type) {
	case middlewareProvider:
		return fn
	case func(http.Handler) http.HandlerFunc:
		return func(next http.Handler) http.Handler { return fn(next) }
	case func(http.ResponseWriter, *http.Request):
		return func(next http.Handler) http.Handler {
			// An endpoint with no successor needs no status wrapper. This
			// is native net/http dispatch on top of chi's route match.
			if next == nil {
				return http.HandlerFunc(fn)
			}
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status, ok := w.(types.ResponseStatusProvider)
				if !ok {
					tracked := &statusWriter{ResponseWriter: w}
					w, status = tracked, tracked
				}
				fn(w, r)
				if status.WrittenStatus() == 0 {
					next.ServeHTTP(w, r)
				}
			})
		}
	}

	value := reflect.ValueOf(h)
	typ := value.Type()
	if typ.Kind() != reflect.Func || value.IsNil() {
		panic(fmt.Sprintf("web: expected handler function; got %T", h))
	}
	providersMu.RLock()
	factory := typedHandlers[typ]
	providersMu.RUnlock()
	if factory != nil {
		return factory(h)
	}
	if typ.NumOut() != 0 {
		panic(fmt.Sprintf("web: handler cannot return values: %s", typ))
	}
	if typ.NumIn() == 0 {
		panic(fmt.Sprintf("web: handler must take request/response context: %s", typ))
	}

	// Validate once during route registration, not by waiting for the first request.
	statusProvided := false
	for i := 0; i < typ.NumIn(); i++ {
		input := typ.In(i)
		switch input {
		case responseWriterType:
			statusProvided = true
		case httpRequestType:
			// *http.Request itself does not tell whether a response was sent.
		default:
			if _, ok := providerFor(input); !ok {
				panic(fmt.Sprintf("web: unregistered handler argument %s in %s", input, typ))
			}
			statusProvided = true
		}
	}
	if !statusProvided {
		panic("web: handler needs a response/status argument: " + typ.String())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap when outer context middleware has not installed its own writer.
			tracked, ok := w.(types.ResponseStatusProvider)
			if !ok {
				w = &statusWriter{ResponseWriter: w}
				tracked = w.(types.ResponseStatusProvider)
			}
			args := make([]reflect.Value, typ.NumIn())
			statuses := []types.ResponseStatusProvider{tracked}
			for i := range args {
				input := typ.In(i)
				switch input {
				case responseWriterType:
					args[i] = reflect.ValueOf(w)
				case httpRequestType:
					args[i] = reflect.ValueOf(r)
				default:
					resolve, _ := providerFor(input)
					ctx := resolve(r)
					if any(ctx) == nil {
						panic("web: typed context resolver returned nil")
					}
					args[i] = reflect.ValueOf(ctx)
					statuses = append(statuses, ctx)
				}
			}
			value.Call(args)
			for _, status := range statuses {
				if status.WrittenStatus() != 0 {
					return
				}
			}
			if next != nil {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// statusWriter is used when a route has no request-scoped response wrapper.
// It lets ordinary HTTP endpoints participate in Gitea-style short-circuiting.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WrittenStatus() int          { return w.status }
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *statusWriter) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (w *statusWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func adaptHandlers(groupMiddleware, afterRouting, endpoints []any) http.HandlerFunc {
	if len(endpoints) == 0 {
		panic("web: endpoint handler required")
	}
	all := make([]any, 0, len(afterRouting)+len(groupMiddleware)+len(endpoints))
	// Pre-middleware executes before standard middleware of any scope.
	for _, group := range [][]any{afterRouting, groupMiddleware, endpoints} {
		for _, m := range group {
			if p, ok := m.(types.PreMiddlewareProvider); ok && p != nil {
				all = append(all, middlewareProvider(p))
			}
		}
	}
	for _, group := range [][]any{afterRouting, groupMiddleware, endpoints} {
		for _, m := range group {
			if _, pre := m.(types.PreMiddlewareProvider); !pre && m != nil {
				all = append(all, m)
			}
		}
	}
	if len(all) == 0 {
		panic("web: no endpoint")
	}
	providers := make([]middlewareProvider, len(all))
	for i, m := range all {
		providers[i] = toHandlerProvider(m)
	}
	// Middleware composition is performed at registration, not on every request.
	var chain http.Handler
	for i := len(providers) - 1; i >= 0; i-- {
		chain = providers[i](chain)
	}
	return chain.ServeHTTP
}
