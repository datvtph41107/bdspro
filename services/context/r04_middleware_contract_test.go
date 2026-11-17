package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
)

type r04ActorKey struct{}

// Authentication in this test is a synthetic guard: it tests Router ordering
// and denial mechanics, NOT an implementation of production authentication.
func TestR04BeforeRoutingAndScopedAuthenticationOrder(t *testing.T) {
	parent := web.NewRouter()
	child := web.NewRouter()

	var trace []string
	parent.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace = append(trace, "ingress")
			if r.URL.Path == "/api/blocked" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	parent.AfterRouting(APIContexterHTTP())
	parent.AfterRouting(func(api *APIContext) {
		trace = append(trace, "parent")
	})

	child.AfterRouting(func(api *APIContext) {
		trace = append(trace, "child")
	})
	child.Group("/private", func() {
		child.Get("/profile", func(api *APIContext) {
			trace = append(trace, "endpoint")
			if got := api.Value(r04ActorKey{}); got != "verified-actor" {
				t.Errorf("actor context=%v", got)
			}
			api.Status(http.StatusNoContent)
		})
	}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			trace = append(trace, "auth")
			if req.Header.Get("Authorization") != "Bearer test-fixture" {
				GetAPIContextFromRequest(req).AuthenticationRequired()
				return
			}
			req = req.WithContext(context.WithValue(req.Context(), r04ActorKey{}, "verified-actor"))
			next.ServeHTTP(w, req)
		})
	})
	parent.Mount("/api", child)

	scenarios := []struct {
		name, path, token string
		status            int
		want              []string
	}{
		{"blocked before matcher", "/api/blocked", "", http.StatusForbidden,
			[]string{"ingress"}},
		{"no credentials", "/api/private/profile", "", http.StatusUnauthorized,
			[]string{"ingress", "parent", "child", "auth"}},
		{"authorized", "/api/private/profile", "Bearer test-fixture", http.StatusNoContent,
			[]string{"ingress", "parent", "child", "auth", "endpoint"}},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			trace = nil
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}
			response := httptest.NewRecorder()
			parent.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Errorf("status=%d want=%d", response.Code, tc.status)
			}
			if !reflect.DeepEqual(trace, tc.want) {
				t.Errorf("trace=%v want=%v", trace, tc.want)
			}
			if tc.status == http.StatusUnauthorized {
				if got := response.Header().Get("WWW-Authenticate"); got != "Bearer" {
					t.Errorf("WWW-Authenticate=%q", got)
				}
				if body := response.Body.String(); body == "" {
					t.Error("denial was not serialized as an API error response")
				}
			}
		})
	}
}

func TestR04TypedGuardAbortWithoutResponse(t *testing.T) {
	m := web.NewRouter()
	m.AfterRouting(APIContexterHTTP())
	reached := false
	m.Group("/restricted", func() {
		m.Get("", func(api *APIContext) {
			reached = true
			api.Status(http.StatusNoContent)
		})
	}, func(api *APIContext) {
		api.Abort()
	})

	response := httptest.NewRecorder()
	m.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/restricted", nil))
	if reached {
		t.Fatal("typed authorization guard aborted but endpoint still ran")
	}
}
