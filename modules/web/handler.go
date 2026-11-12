package web

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/datvtph41107/bdspro/modules/web/types"
)

type middlewareProvider = func(http.Handler) http.Handler

var (
	responseProvidersMu     sync.RWMutex
	responseStatusProviders = map[reflect.Type]func(*http.Request) types.ResponseStatusProvider{}
)

func RegisterResponseStatusProvider[T any](fn func(*http.Request) types.ResponseStatusProvider) {
	if fn == nil {
		panic("web: nil response status provider")
	}
	responseProvidersMu.Lock()
	defer responseProvidersMu.Unlock()
	responseStatusProviders[reflect.TypeFor[T]()] = fn
}

var (
	requestType = reflect.TypeFor[*http.Request]()
	writerType  = reflect.TypeFor[http.ResponseWriter]()
	statusType  = reflect.TypeFor[types.ResponseStatusProvider]()
)

func isNilOrFuncNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	return v.Kind() == reflect.Func && v.IsNil()
}

func prepareHandleArgsIn(w http.ResponseWriter, req *http.Request, fn reflect.Value) []reflect.Value {
	args := make([]reflect.Value, fn.Type().NumIn())
	for i := range args {
		t := fn.Type().In(i)
		switch t {
		case requestType:
			args[i] = reflect.ValueOf(req)
		case writerType:
			args[i] = reflect.ValueOf(w)
		default:
			responseProvidersMu.RLock()
			provider := responseStatusProviders[t]
			responseProvidersMu.RUnlock()
			if provider == nil {
				panic(fmt.Sprintf("web: unsupported handler argument %s", t))
			}
			value := provider(req)
			if value == nil {
				panic(fmt.Sprintf("web: provider for %s returned nil", t))
			}
			v := reflect.ValueOf(value)
			if !v.Type().AssignableTo(t) {
				panic(fmt.Sprintf("web: provider returned %s, expected %s", v.Type(), t))
			}
			args[i] = v
		}
	}
	return args
}

func preCheckHandler(fn reflect.Value) {
	t := fn.Type()
	if t.NumOut() != 0 {
		panic(fmt.Sprintf("web: handler must not return values: %s", t))
	}
	hasStatus := false
	for i := 0; i < t.NumIn(); i++ {
		arg := t.In(i)
		if arg == writerType {
			hasStatus = true
			continue
		}
		if arg == requestType {
			continue
		}
		responseProvidersMu.RLock()
		provider := responseStatusProviders[arg]
		responseProvidersMu.RUnlock()
		if provider == nil {
			panic(fmt.Sprintf("web: unsupported handler argument %s", arg))
		}
		if !arg.Implements(statusType) {
			panic(fmt.Sprintf("web: argument %s must report response status", arg))
		}
		hasStatus = true
	}
	if !hasStatus {
		panic(fmt.Sprintf("web: handler needs response status provider: %s", t))
	}
}

func hasResponseBeenWritten(w http.ResponseWriter, args []reflect.Value) bool {
	if status, ok := w.(types.ResponseStatusProvider); ok && status.WrittenStatus() != 0 {
		return true
	}
	for _, value := range args {
		if provider, ok := value.Interface().(types.ResponseStatusProvider); ok && provider.WrittenStatus() != 0 {
			return true
		}
	}
	return false
}

// toHandlerProvider converts HTTP providers and typed actions to the
// same net/http continuation contract used by Gitea.
func toHandlerProvider(handler any) middlewareProvider {
	if isNilOrFuncNil(handler) {
		panic("web: nil handler")
	}
	switch h := handler.(type) {
	case types.PreMiddlewareProvider:
		return middlewareProvider(h)
	case func(http.Handler) http.Handler:
		return h
	case func(http.Handler) http.HandlerFunc:
		return func(next http.Handler) http.Handler { return h(next) }
	}
	fn := reflect.ValueOf(handler)
	if fn.Kind() != reflect.Func {
		panic(fmt.Sprintf("web: handler must be a function: %T", handler))
	}
	preCheckHandler(fn)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			args := prepareHandleArgsIn(w, req, fn)
			fn.Call(args)
			if next != nil && !hasResponseBeenWritten(w, args) {
				next.ServeHTTP(w, req)
			}
		})
	}
}

func appendPre(providers []middlewareProvider, actions []any) []middlewareProvider {
	for _, action := range actions {
		if pre, ok := action.(types.PreMiddlewareProvider); ok && pre != nil {
			providers = append(providers, toHandlerProvider(pre))
		}
	}
	return providers
}

func appendNormal(providers []middlewareProvider, actions []any) []middlewareProvider {
	for _, action := range actions {
		if _, ok := action.(types.PreMiddlewareProvider); !ok && !isNilOrFuncNil(action) {
			providers = append(providers, toHandlerProvider(action))
		}
	}
	return providers
}

func wrapMiddlewareAndHandler(global, group, actions []any) ([]middlewareProvider, http.HandlerFunc) {
	if len(actions) == 0 || isNilOrFuncNil(actions[len(actions)-1]) {
		panic("web: route endpoint is required")
	}
	if _, ok := actions[len(actions)-1].(types.PreMiddlewareProvider); ok {
		panic("web: final endpoint cannot be pre-middleware")
	}
	providers := make([]middlewareProvider, 0, len(global)+len(group)+len(actions))
	providers = appendPre(providers, global)
	providers = appendPre(providers, group)
	providers = appendPre(providers, actions)
	providers = appendNormal(providers, global)
	providers = appendNormal(providers, group)
	providers = appendNormal(providers, actions)
	last := providers[len(providers)-1]
	return providers[:len(providers)-1], last(nil).ServeHTTP
}

func composeHandler(middlewares []middlewareProvider, endpoint http.HandlerFunc) http.Handler {
	var handler http.Handler = endpoint
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
