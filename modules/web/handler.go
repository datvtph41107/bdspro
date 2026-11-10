package web

import (
	"fmt"
	"net/http"
	"reflect"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

// responseStatusProviders resolves typed actions from the request after
// APIContexter has installed the request-local context. Registrations occur
// during package initialization, before routes are registered.
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

// preCheckHandler validates typed action signatures at registration, not
// on the first request. Each action must expose a response status provider.
func preCheckHandler(typ reflect.Type) {
	if typ.NumOut() != 0 {
		panic(fmt.Sprintf("web: handler must not return values: %s", typ))
	}
	hasProvider := false
	for i := 0; i < typ.NumIn(); i++ {
		arg := typ.In(i)
		switch arg {
		case reflect.TypeFor[*http.Request](), reflect.TypeFor[http.ResponseWriter]():
		default:
			if _, ok := responseStatusProviders[arg]; !ok {
				panic(fmt.Sprintf("web: unsupported handler argument %s", arg))
			}
			hasProvider = true
		}
	}
	if !hasProvider {
		panic(fmt.Sprintf("web: handler must receive a ResponseStatusProvider: %s", typ))
	}
}

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

// toHandlerProvider converts HTTP middleware and registered typed actions
// into the same continuation contract. Validation is performed at registration.
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
		fn := reflect.ValueOf(handler)
		if !fn.IsValid() || fn.Kind() != reflect.Func || fn.IsNil() {
			panic(fmt.Sprintf("web: unsupported HTTP handler %T", handler))
		}
		typ := fn.Type()
		preCheckHandler(typ)
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				tracked, ok := w.(*responseWriter)
				if !ok {
					tracked = &responseWriter{ResponseWriter: w}
				}
				args := make([]reflect.Value, typ.NumIn())
				providers := make([]web_types.ResponseStatusProvider, 0, typ.NumIn())
				for i := range args {
					arg := typ.In(i)
					switch arg {
					case reflect.TypeFor[*http.Request]():
						args[i] = reflect.ValueOf(req)
					case reflect.TypeFor[http.ResponseWriter]():
						args[i] = reflect.ValueOf(tracked)
					default:
						provider := responseStatusProviders[arg](req)
						if provider == nil {
							panic(fmt.Sprintf("web: nil response status provider for %s", arg))
						}
						v := reflect.ValueOf(provider)
						if !v.Type().AssignableTo(arg) {
							panic(fmt.Sprintf("web: provider %s is incompatible with %s", v.Type(), arg))
						}
						args[i] = v
						providers = append(providers, provider)
					}
				}
				fn.Call(args)
				written := tracked.WrittenStatus() != 0
				for _, provider := range providers {
					written = written || provider.WrittenStatus() != 0
				}
				if next != nil && !written {
					next.ServeHTTP(tracked, req)
				}
			})
		}
	}
}
