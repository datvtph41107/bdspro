package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRouterGroupsAndMethods(t *testing.T) {
	r := NewRouter()
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Group("/organizations", func() {
		r.Get("", func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(204) })
		r.Group("/{id}", func() {
			r.Get("", func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(202) })
			r.Patch("", func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(200) })
		})
	})
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/health", 200}, {"GET", "/organizations", 204},
		{"GET", "/organizations/7", 202}, {"PATCH", "/organizations/7", 200},
		{"POST", "/organizations/7", 405},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status {
			t.Errorf("%s %s: status %d want %d", tc.method, tc.path, w.Code, tc.status)
		}
	}
}

func TestBeforeAfterRoutingAndShortCircuit(t *testing.T) {
	r := NewRouter()
	order := []string{}
	before := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "before")
			next.ServeHTTP(w, req)
		})
	}
	after := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { order = append(order, "after"); next.ServeHTTP(w, req) })
	}
	r.BeforeRouting(before)
	r.AfterRouting(after)
	r.Group("/organizations", func() {
		r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) { order = append(order, "endpoint"); w.WriteHeader(200) })
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/organizations/4", nil))
	if w.Code != 200 || !reflect.DeepEqual(order, []string{"before", "after", "endpoint"}) {
		t.Fatalf("status=%d order=%v", w.Code, order)
	}
}

func BenchmarkRouterSimple(b *testing.B) {
	r := NewRouter()
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(204) })
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(httptest.NewRecorder(), req)
	}
}
