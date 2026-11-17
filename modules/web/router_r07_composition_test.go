package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

// R07 guards must inherit from parent groups and must not leak to sibling routes.
func TestR07NestedGroupMountScopesAndSiblingIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root, child := NewRouter(), NewRouter()
	var trace []string

	guard := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				trace = append(trace, name)
				next.ServeHTTP(w, req)
			})
		}
	}
	endpoint := func(name string) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, _ *http.Request) {
			trace = append(trace, name)
			w.WriteHeader(http.StatusNoContent)
		}
	}

	root.AfterRouting(guard("root"))
	child.AfterRouting(guard("child"))
	child.Group("/{team}", func() {
		child.Get("/view", endpoint("team-endpoint"))
	}, guard("team"))

	root.Group("/orgs/{org}", func() {
		root.Mount("/teams", child)
		root.Get("/profile", endpoint("org-endpoint"))
	}, guard("org"))
	root.Get("/public", endpoint("public-endpoint"))

	cases := []struct {
		path string
		want []string
	}{
		{"/orgs/acme/teams/42/view", []string{"root", "org", "child", "team", "team-endpoint"}},
		{"/orgs/acme/profile", []string{"root", "org", "org-endpoint"}},
		{"/public", []string{"root", "public-endpoint"}},
	}
	for _, tc := range cases {
		trace = nil
		rec := httptest.NewRecorder()
		root.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusNoContent || !reflect.DeepEqual(trace, tc.want) {
			t.Errorf("%s: status=%d trace=%v want=%v", tc.path, rec.Code, trace, tc.want)
		}
	}
}

// A mounted child's Gin route table has been flattened into its parent.
// Any later route registration must fail loudly rather than claim a route
// that parent requests will never reach.
func TestR07MountedChildRejectsLateRouteAndMiddlewareMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	parent, child := NewRouter(), NewRouter()
	child.Get("/existing", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	parent.Mount("/api", child)

	noop := func(next http.Handler) http.Handler { return next }
	mutations := []struct {
		name string
		run  func()
	}{
		{"Methods", func() {
			child.Get("/late", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		}},
		{"Group", func() { child.Group("/late", func() {}) }},
		{"BeforeRouting", func() { child.BeforeRouting(noop) }},
		{"AfterRouting", func() { child.AfterRouting(noop) }},
		{"Mount", func() {
			grandchild := NewRouter()
			grandchild.Get("/child", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
			child.Mount("/late", grandchild)
		}},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("mounted router accepted late mutation")
				}
			}()
			tc.run()
		})
	}
	rec := httptest.NewRecorder()
	parent.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/existing", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("mounted route status=%d", rec.Code)
	}
}

// Reusing one frozen child at two mount points is safe and keeps the
// route descriptors identical; neither mount permits late child mutation.
func TestR07FrozenChildCanMountTwice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root, child := NewRouter(), NewRouter()
	child.Get("/ping", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	root.Mount("/one", child)
	root.Mount("/two", child)
	for _, path := range []string{"/one/ping", "/two/ping"} {
		rec := httptest.NewRecorder()
		root.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("%s status=%d", path, rec.Code)
		}
	}
}
