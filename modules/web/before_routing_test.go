package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web/types"
)

func TestBeforeRoutingRewritesPathBeforeGinMatch(t *testing.T) {
	router := NewRouter()
	var before, endpoint bool
	router.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			before = true
			req.URL.Path = "/correct"
			next.ServeHTTP(w, req)
		})
	})
	router.Get("/correct", func(w http.ResponseWriter, req *http.Request) {
		endpoint = true
		w.WriteHeader(http.StatusNoContent)
	})
	out := httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/wrong", nil))
	if !before || !endpoint || out.Code != http.StatusNoContent {
		t.Fatalf("before=%v endpoint=%v status=%d", before, endpoint, out.Code)
	}
}

func TestBeforeRoutingPreservesRequestContext(t *testing.T) {
	router := NewRouter()
	type key struct{}
	router.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), key{}, "injected")))
		})
	})
	router.Get("/test", func(w http.ResponseWriter, req *http.Request) {
		if req.Context().Value(key{}) != "injected" {
			t.Fatal("before-routing context value was lost")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	out := httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/test", nil))
	if out.Code != http.StatusNoContent {
		t.Fatalf("status=%d", out.Code)
	}
}

func TestBeforeRoutingRejectsTypedAction(t *testing.T) {
	router := NewRouter()
	defer func() {
		if recover() == nil {
			t.Fatal("typed API middleware cannot execute before route match")
		}
	}()
	router.BeforeRouting(func(api *testAPI) {})
}

func TestMountedScopeOrdering(t *testing.T) {
	var order []string
	mark := func(label string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, label)
				next.ServeHTTP(w, req)
			})
		}
	}
	root := NewRouter()
	child := NewRouter()
	root.AfterRouting(mark("parent-normal"))
	child.AfterRouting(types.PreMiddlewareProvider(mark("child-pre")), mark("child-normal"))
	child.Get("/item", func(w http.ResponseWriter, req *http.Request) {
		order = append(order, "endpoint")
		w.WriteHeader(http.StatusNoContent)
	})
	root.Mount("/api", child)
	w := httptest.NewRecorder()
	root.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/item", nil))
	want := []string{"parent-normal", "child-pre", "child-normal", "endpoint"}
	if w.Code != http.StatusNoContent || !reflect.DeepEqual(order, want) {
		t.Fatalf("status=%d order=%v want=%v", w.Code, order, want)
	}
}

func TestBeforeRoutingExecutesOnNotFound(t *testing.T) {
	root := NewRouter()
	called := false
	root.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			called = true
			next.ServeHTTP(w, req)
		})
	})
	w := httptest.NewRecorder()
	root.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if !called || w.Code != http.StatusNotFound {
		t.Fatalf("called=%v status=%d", called, w.Code)
	}
}
