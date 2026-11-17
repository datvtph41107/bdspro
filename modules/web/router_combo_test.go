package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGiteaComboSharedActionsAndInheritedGroupMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewRouter()
	trace := []string{}
	guard := func(label string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				trace = append(trace, label)
				next.ServeHTTP(w, r)
			})
		}
	}
	endpoint := func(label string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			trace = append(trace, label)
			w.WriteHeader(http.StatusNoContent)
		}
	}
	m.AfterRouting(guard("after"))
	m.Group("/orgs/{org}", func() {
		m.Combo("/teams/{team}", guard("shared")).
			Get(endpoint("get")).
			Post(endpoint("post")).
			Patch(endpoint("patch"))
	}, guard("group"))
	m.Get("/public", endpoint("public"))
	for _, tc := range []struct {
		method, path string
		want []string
	}{
		{http.MethodGet, "/orgs/acme/teams/42", []string{"after", "group", "shared", "get"}},
		{http.MethodPost, "/orgs/acme/teams/42", []string{"after", "group", "shared", "post"}},
		{http.MethodPatch, "/orgs/acme/teams/42", []string{"after", "group", "shared", "patch"}},
		{http.MethodGet, "/public", []string{"after", "public"}},
	} {
		trace = nil
		rec := httptest.NewRecorder()
		m.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusNoContent || !reflect.DeepEqual(trace, tc.want) {
			t.Errorf("%s %s: status=%d trace=%v want=%v", tc.method, tc.path, rec.Code, trace, tc.want)
		}
	}
}

func TestGiteaComboRejectsLateMethodsOnMountedChild(t *testing.T) {
	gin.SetMode(gin.TestMode)
	parent, child := NewRouter(), NewRouter()
	child.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	parent.Mount("/api", child)
	defer func() {
		if recover() == nil {
			t.Fatal("Combo accepted registration after child mounted")
		}
	}()
	child.Combo("/late").Post(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}
