package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

)

func TestGiteaParameterGroupAndMountedRoute(t *testing.T) {

	parent := NewRouter()
	child := NewRouter()

	var order []string
	guard := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, req)
			})
		}
	}
	parent.AfterRouting(guard("parent"))
	child.AfterRouting(guard("child"))
	child.Group("/teams", func() {
		child.Get("/{team_id}", func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "endpoint")
			if req.PathValue("org") != "example" || req.PathValue("team_id") != "42" {
				t.Errorf("route parameters not available through matched request")
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}, guard("group"))

	parent.Mount("/orgs/{org}", child)

	rec := httptest.NewRecorder()
	parent.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orgs/example/teams/42", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204", rec.Code)
	}
	if want := []string{"parent", "child", "group", "endpoint"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("middleware order=%v, want %v", order, want)
	}
}

func TestGiteaParameterDirectRouteAndLegacyGinSyntax(t *testing.T) {
	r := NewRouter()
	r.Group("/organizations/{org}", func() {
		r.Get("/teams/{team}", func(w http.ResponseWriter, req *http.Request) {
			if req.PathValue("org") != "acme" || req.PathValue("team") != "backend" {
				t.Errorf("wrong parameter values: org=%q team=%q", req.PathValue("org"), req.PathValue("team"))
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
	r.Get("/legacy/:id", func(w http.ResponseWriter, req *http.Request) {
		if req.PathValue("id") != "7" {
			t.Errorf("legacy param=%q", req.PathValue("id"))
		}
		w.WriteHeader(http.StatusNoContent)
	})

	for _, path := range []string{"/organizations/acme/teams/backend", "/legacy/7"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNoContent {
			t.Errorf("path %s: status=%d, want 204", path, rec.Code)
		}
	}
}

func TestChiPatternPreservesGiteaSyntax(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
	}{
		{"/orgs/{org}", "/orgs/{org}"},
		{"/orgs/{org:[a-z]+}", "/orgs/{org:[a-z]+}"},
		{"/legacy/:id", "/legacy/{id}"},
		{"/swagger/*filepath", "/swagger/*"},
	} {
		if got := chiPattern(tc.path); got != tc.want {
			t.Errorf("chiPattern(%q)=%q want=%q", tc.path, got, tc.want)
		}
	}
}
