package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestChiRouterMatchMiddlewareMount(t *testing.T) {
	parent, child := NewRouter(), NewRouter()
	var calls []string
	parent.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			calls = append(calls, "ingress")
			next.ServeHTTP(w, req)
		})
	})
	parent.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			calls = append(calls, "parent")
			next.ServeHTTP(w, req)
		})
	})
	child.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			calls = append(calls, "child")
			next.ServeHTTP(w, req)
		})
	})
	child.Get("/teams/{id}", func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, "endpoint")
		if req.PathValue("org") != "acme" || req.PathValue("id") != "42" {
			t.Errorf("path values org=%q id=%q", req.PathValue("org"), req.PathValue("id"))
		}
		w.WriteHeader(http.StatusNoContent)
	})
	parent.Mount("/orgs/{org}", child)
	w := httptest.NewRecorder()
	parent.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orgs/acme/teams/42", nil))
	if w.Code != http.StatusNoContent {
		t.Errorf("status=%d", w.Code)
	}
	if want := []string{"ingress", "parent", "child", "endpoint"}; !reflect.DeepEqual(calls, want) {
		t.Errorf("calls=%v want=%v", calls, want)
	}
}

func TestChiRouterTrustedClientIP(t *testing.T) {
	r := NewRouter()
	if err := r.SetTrustedProxies([]string{"127.0.0.1/32"}); err != nil {
		t.Fatal(err)
	}
	r.Get("/ip", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(ClientIPFromRequest(req)))
	})
	for _, tc := range []struct {
		remote, forwarded, want string
	}{
		{"127.0.0.1:1000", "203.0.113.9", "203.0.113.9"},
		{"192.0.2.10:1000", "203.0.113.9", "192.0.2.10"},
		{"127.0.0.1:1000", "198.51.100.5, 203.0.113.9", "203.0.113.9"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ip", nil)
		req.RemoteAddr = tc.remote
		req.Header.Set("X-Forwarded-For", tc.forwarded)
		r.ServeHTTP(rec, req)
		if rec.Body.String() != tc.want {
			t.Errorf("remote=%s forwarded=%q resolved=%q want=%q", tc.remote, tc.forwarded, rec.Body.String(), tc.want)
		}
	}
}
