package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/gin-gonic/gin"
)

// This synthetic resource fixture is ONLY for testing middleware composition.
// BDSPro does not yet have a production organization resource/context contract.
func TestR06MountedResourceAssignmentAndPermissionPreconditions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	parent := web.NewRouter()
	child := web.NewRouter()
	parent.AfterRouting(APIContexter())

	ingressCalls, lookupCalls, endpointCalls := 0, 0, 0
	child.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ingressCalls++
			next.ServeHTTP(w, req)
		})
	})

	// A typed action models a Gitea-style resource assignment middleware:
	// it runs after APIContexter and before the endpoint, and can stop the chain.
	assign := func(api *APIContext) {
		lookupCalls++
		org := api.PathParam("org")
		if org == "missing" {
			api.APIErrorNotFound()
			return
		}
		api.GetData()["r06.org"] = org
	}
	permission := func(api *APIContext) {
		if api.GetData()["r06.org"] != "acme" {
			api.APIError(http.StatusForbidden, "permission_denied", "permission denied")
		}
	}
	child.Group("/teams", func() {
		child.Get("/{team}", func(api *APIContext) {
			endpointCalls++
			if got := api.GetData()["r06.org"]; got != "acme" {
				t.Errorf("endpoint resource=%v", got)
			}
			if got := api.PathParam("team"); got != "42" {
				t.Errorf("team path param=%q", got)
			}
			api.Status(http.StatusNoContent)
		})
	}, assign, permission)
	parent.Mount("/orgs/{org}", child)

	cases := []struct {
		path string
		want int
	}{
		{"/orgs/acme/teams/42", http.StatusNoContent},
		{"/orgs/other/teams/42", http.StatusForbidden},
		{"/orgs/missing/teams/42", http.StatusNotFound},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		parent.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.want {
			t.Errorf("GET %s status=%d want=%d body=%s", tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
	if ingressCalls != 3 || lookupCalls != 3 || endpointCalls != 1 {
		t.Fatalf("ingress=%d lookups=%d endpoints=%d want=3/3/1", ingressCalls, lookupCalls, endpointCalls)
	}
}
