package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web"
)

// H01 protects the HTTP contracts that must survive swapping Gin for Chi.
// This test deliberately imports neither Gin nor Chi; the same application
// router declarations must continue working after the matcher migration.
func TestH01HTTPBoundaryContract(t *testing.T) {
	type actorKey struct{}
	parent, child := web.NewRouter(), web.NewRouter()
	var calls []string
	cleanups := 0
	parent.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			calls = append(calls, "ingress")
			req = req.WithContext(context.WithValue(req.Context(), actorKey{}, "ingress"))
			next.ServeHTTP(w, req)
		})
	})
	parent.AfterRouting(APIContexterHTTP())
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
	child.Get("/teams/{id}", func(api *APIContext) {
		calls = append(calls, "handler")
		if got := api.Value(actorKey{}); got != "ingress" {
			t.Errorf("context actor=%v", got)
		}
		if got := api.PathParam("org"); got != "acme" {
			t.Errorf("org parameter=%q", got)
		}
		if got, err := api.PositivePathID("id"); got != 42 || err != nil {
			t.Errorf("id=%d err=%v", got, err)
		}
		if GetAPIContextFromRequest(api.CurrentRequest()) != api {
			t.Error("typed handler and request did not share APIContext")
		}
		store := reqctx.FromContext(api.CurrentRequest().Context())
		if store == nil {
			t.Fatal("request scope missing")
		}
		store.AddCleanup(func() { cleanups++ })
		api.JSON(http.StatusOK, map[string]bool{"ok": true})
	}, func(*APIContext) {
		t.Error("handler continued after committed HTTP response")
	})
	parent.Mount("/orgs/{org}", child)

	rec := httptest.NewRecorder()
	parent.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orgs/acme/teams/42", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"ok\":true}" {
		t.Fatalf("response status=%d body=%q", rec.Code, rec.Body.String())
	}
	if cleanups != 1 {
		t.Errorf("cleanup calls=%d, want one", cleanups)
	}
	if want := []string{"ingress", "parent", "child", "handler"}; !reflect.DeepEqual(calls, want) {
		t.Errorf("middleware order=%v want=%v", calls, want)
	}
}

func TestH01HTTPBoundaryUnmatchedRouteDoesNotCreateAPIContext(t *testing.T) {
	router := web.NewRouter()
	called := false
	router.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			called = true
			next.ServeHTTP(w, req)
		})
	})
	router.AfterRouting(APIContexterHTTP())
	router.Get("/health", func(api *APIContext) {
		api.Status(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/not-found", nil))
	if rec.Code != http.StatusNotFound || !called {
		t.Fatalf("unmatched route status=%d ingress called=%v", rec.Code, called)
	}
}
