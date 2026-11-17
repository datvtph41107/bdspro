package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

type missingProviderContext struct{}

func (*missingProviderContext) WrittenStatus() int { return 0 }

func TestTypedProviderRejectsTypedNilBeforeHandlerInvocation(t *testing.T) {
	// Register the test-only provider as package initialization would register
	// an APIContext provider. Use the existing registry contract.
	RegisterResponseStatusProvider[*missingProviderContext](
		func(*http.Request) web_types.ResponseStatusProvider {
			var missing *missingProviderContext
			return missing
		},
	)
	called := false
	provider := toHandlerProvider(func(*missingProviderContext) {
		called = true
	})
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("typed nil provider reached the handler")
		}
		if !strings.Contains(r.(string), "typed nil response status provider") {
			t.Fatalf("panic=%v", r)
		}
		if called {
			t.Fatal("handler invoked with typed nil context")
		}
	}()
	provider(nil).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestTypedHandlerPrecheckRejectsUnsupportedSignatures(t *testing.T) {
	for _, handler := range []any{
		func(*http.Request) {},
		func(http.ResponseWriter) error { return nil },
		func(int) {},
	} {
		t.Run(reflect.TypeOf(handler).String(), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("unsupported action %T accepted at registration", handler)
				}
			}()
			toHandlerProvider(handler)
		})
	}
}
