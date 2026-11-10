package web

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"reflect"

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
