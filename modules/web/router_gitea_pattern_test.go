package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGiteaParameterGroupAndMountedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
			c := GinContextFromRequest(req)
			if c == nil || c.Param("org") != "example" || c.Param("team_id") != "42" {
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
	gin.SetMode(gin.TestMode)
	r := NewRouter()
	r.Group("/organizations/{org}", func() {
		r.Get("/teams/{team}", func(w http.ResponseWriter, req *http.Request) {
			c := GinContextFromRequest(req)
			if c.Param("org") != "acme" || c.Param("team") != "backend" {
				t.Errorf("wrong parameter values: org=%q team=%q", c.Param("org"), c.Param("team"))
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
	r.Get("/legacy/:id", func(w http.ResponseWriter, req *http.Request) {
		if c := GinContextFromRequest(req); c.Param("id") != "7" {
			t.Errorf("legacy Gin param=%q", c.Param("id"))
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

func TestGiteaParameterRejectsUnsupportedChiPatterns(t *testing.T) {
	for _, pattern := range []string{
		"/{id:[0-9]+}", "/{name}.json", "/{bad-name}",
		"/{id", "/id}", "/{}", "/{9id}", "/{x}/{y:one|two}",
	} {
		t.Run(pattern, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("unsupported pattern %q was accepted", pattern)
				}
			}()
			ginPattern(pattern)
		})
	}
}
