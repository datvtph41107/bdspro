package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web/types"
)

type testAPI struct {
	Status int
	Store  reqctx.RequestContext
}

func (a *testAPI) WrittenStatus() int { return a.Status }

type testKey struct{}

func init() {
	RegisterResponseStatusProvider[*testAPI](func(req *http.Request) types.ResponseStatusProvider {
		return reqctx.MustContextValue[*testAPI](req.Context(), testKey{})
	})
}

func TestGroupAndPreMiddlewareOrder(t *testing.T) {
	r := NewRouter()
	var order []string
	init := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			store := reqctx.FromContext(req.Context())
			store.SetContextValue(testKey{}, &testAPI{Store: store})
			next.ServeHTTP(w, req)
		})
	}
	marker := func(s string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, s)
				next.ServeHTTP(w, req)
			})
		}
	}
	r.AfterRouting(init, marker("global"))
	r.Group("/users", func() {
		r.Get("/{name}", types.PreMiddlewareProvider(marker("pre")), marker("route"), func(api *testAPI) {
			order = append(order, "endpoint")
		})
	}, marker("group"))
	req := httptest.NewRequest(http.MethodGet, "/users/one", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	want := []string{"pre", "global", "group", "route", "endpoint"}
	// init is a normal middleware and runs after pre, before endpoint.
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order=%v want=%v", order, want)
	}
}

func TestMountAndCleanup(t *testing.T) {
	child := NewRouter()
	root := NewRouter()
	root.Mount("/api/v1", child)
	child.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		store := reqctx.FromContext(req.Context())
		if store == nil {
			t.Fatal("request datastore missing")
		}
		store.AddCleanup(func() { w.Header().Set("X-Cleanup", "done") })
		w.WriteHeader(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	root.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHandlerRegistrationRejectsInvalidSignature(t *testing.T) {
	r := NewRouter()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	r.Get("/bad", func() {})
}

func TestRequestContextPropagation(t *testing.T) {
	r := NewRouter()
	type key struct{}
	parent := context.WithValue(context.Background(), key{}, "value")
	r.Get("/x", func(w http.ResponseWriter, req *http.Request) {
		if req.Context().Value(key{}) != "value" {
			t.Fatal("request value lost")
		}
		w.WriteHeader(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(parent))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}
