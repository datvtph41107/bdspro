package web

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"slices"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

// Registrations happen at package initialization, before serving requests.
// The request-scoped provider is resolved only when the handler executes.
var responseStatusProviders = map[reflect.Type]func(*http.Request) web_types.ResponseStatusProvider{}

func RegisterResponseStatusProvider[T any](provider func(*http.Request) web_types.ResponseStatusProvider) {
	if provider == nil {
		panic("web: nil response status provider")
	}
	typ := reflect.TypeFor[T]()
	if _, exists := responseStatusProviders[typ]; exists {
		panic(fmt.Sprintf("web: duplicate response status provider for %s", typ))
	}
	responseStatusProviders[typ] = provider
}

func isRegisteredTypedHandler(handler any) bool {
	typ := reflect.TypeOf(handler)
	if typ == nil || typ.Kind() != reflect.Func {
		return false
	}
	for i := 0; i < typ.NumIn(); i++ {
		if _, ok := responseStatusProviders[typ.In(i)]; ok {
			return true
		}
	}
	return false
}

// responseWriter observes actual HTTP commitment, not merely a selected status.
type responseWriter struct {
	http.ResponseWriter
	status int
}

var _ web_types.ResponseStatusProvider = (*responseWriter)(nil)

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

// Hijack follows Gitea's responseWriter contract so HTTP handlers that
// explicitly require a websocket/connection upgrade can retain that
// capability when the underlying writer supports it.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

var (
	httpReqType    = reflect.TypeFor[*http.Request]()
	respWriterType = reflect.TypeFor[http.ResponseWriter]()
)

// preCheckHandler rejects unsupported signatures during route registration.
// Both the HTTP writer and registered typed contexts supply response status.
func preCheckHandler(typ reflect.Type) {
	if typ.NumOut() != 0 {
		panic(fmt.Sprintf("web: handler must not return values: %s", typ))
	}
	hasStatusProvider := false
	for i := 0; i < typ.NumIn(); i++ {
		arg := typ.In(i)
		switch arg {
		case httpReqType:
		case respWriterType:
			hasStatusProvider = true
		default:
			if _, ok := responseStatusProviders[arg]; !ok {
				panic(fmt.Sprintf("web: unsupported handler argument %s", arg))
			}
			hasStatusProvider = true
		}
	}
	if !hasStatusProvider {
		panic(fmt.Sprintf("web: handler must receive a ResponseStatusProvider: %s", typ))
	}
}

// prepareHandleArgsIn is the only reflection-based argument resolution path.
func prepareHandleArgsIn(resp http.ResponseWriter, req *http.Request, typ reflect.Type) []reflect.Value {
	args := make([]reflect.Value, typ.NumIn())
	for i := range args {
		arg := typ.In(i)
		switch arg {
		case respWriterType:
			args[i] = reflect.ValueOf(resp)
		case httpReqType:
			args[i] = reflect.ValueOf(req)
		default:
			provider := responseStatusProviders[arg](req)
			if provider == nil {
				panic(fmt.Sprintf("web: nil response status provider for %s", arg))
			}
			value := reflect.ValueOf(provider)
			if !value.Type().AssignableTo(arg) {
				panic(fmt.Sprintf("web: provider %s is incompatible with %s", value.Type(), arg))
			}
			args[i] = value
		}
	}
	return args
}

func hasResponseBeenWritten(args []reflect.Value) bool {
	for _, arg := range args {
		if status, ok := arg.Interface().(web_types.ResponseStatusProvider); ok && status.WrittenStatus() != 0 {
			return true
		}
	}
	return false
}

type middlewareProvider = func(next http.Handler) http.Handler

// executeMiddlewaresHandler composes providers in reverse so the first
// registered middleware is the first one entered. This is Gitea's
// continuation convention; an action that commits a response stops the chain.
func executeMiddlewaresHandler(w http.ResponseWriter, req *http.Request, middlewares []middlewareProvider, endpoint http.HandlerFunc) {
	handler := http.Handler(endpoint)
	for _, middleware := range slices.Backward(middlewares) {
		handler = middleware(handler)
	}
	handler.ServeHTTP(w, req)
}

func isNilOrFuncNil(handler any) bool {
	if handler == nil {
		return true
	}
	value := reflect.ValueOf(handler)
	return value.Kind() == reflect.Func && value.IsNil()
}

func wrapMiddlewareAppendPre(all []middlewareProvider, handlers []any) []middlewareProvider {
	for _, handler := range handlers {
		if pre, ok := handler.(web_types.PreMiddlewareProvider); ok && pre != nil {
			all = append(all, toHandlerProvider(middlewareProvider(pre)))
		}
	}
	return all
}

func wrapMiddlewareAppendNormal(all []middlewareProvider, handlers []any) []middlewareProvider {
	for _, handler := range handlers {
		if _, pre := handler.(web_types.PreMiddlewareProvider); !pre && !isNilOrFuncNil(handler) {
			all = append(all, toHandlerProvider(handler))
		}
	}
	return all
}

// wrapMiddlewareAndHandler is the single matched-route HTTP compiler.
// Pre providers run ahead of ordinary middleware from the same scopes;
// the final non-nil action is the endpoint and cannot be a pre middleware.
func wrapMiddlewareAndHandler(useMiddlewares, curMiddlewares, handlers []any) ([]middlewareProvider, http.HandlerFunc, bool) {
	if len(handlers) == 0 {
		panic("web: no endpoint handler provided")
	}
	if isNilOrFuncNil(handlers[len(handlers)-1]) {
		panic("web: endpoint handler cannot be nil")
	}
	if _, pre := handlers[len(handlers)-1].(web_types.PreMiddlewareProvider); pre {
		panic("web: endpoint cannot be pre middleware")
	}
	providers := make([]middlewareProvider, 0, len(useMiddlewares)+len(curMiddlewares)+len(handlers))
	providers = wrapMiddlewareAppendPre(providers, useMiddlewares)
	providers = wrapMiddlewareAppendPre(providers, curMiddlewares)
	providers = wrapMiddlewareAppendPre(providers, handlers)
	hasPre := len(providers) != 0
	providers = wrapMiddlewareAppendNormal(providers, useMiddlewares)
	providers = wrapMiddlewareAppendNormal(providers, curMiddlewares)
	providers = wrapMiddlewareAppendNormal(providers, handlers)
	if len(providers) == 0 {
		panic("web: no compilable endpoint handler")
	}
	last := len(providers) - 1
	return providers[:last], providers[last](nil).ServeHTTP, hasPre
}

// toHandlerProvider is framework-independent. Gin integration belongs to
// modules/web/router.go, while action invocation follows Gitea's one path.
func toHandlerProvider(handler any) middlewareProvider {
	fn := reflect.ValueOf(handler)
	if !fn.IsValid() || fn.Kind() != reflect.Func || fn.IsNil() {
		panic(fmt.Sprintf("web: handler must be a non-nil function, got %T", handler))
	}
	if middleware, ok := handler.(middlewareProvider); ok {
		return middleware
	}
	if middleware, ok := handler.(func(http.Handler) http.HandlerFunc); ok {
		return func(next http.Handler) http.Handler { return middleware(next) }
	}

	typ := fn.Type()
	preCheckHandler(typ)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			resp := w
			if _, ok := resp.(web_types.ResponseStatusProvider); !ok {
				resp = &responseWriter{ResponseWriter: resp}
			}
			args := prepareHandleArgsIn(resp, req, typ)
			fn.Call(args)
			if next != nil && !hasResponseBeenWritten(args) {
				next.ServeHTTP(resp, req)
			}
		})
	}
}
