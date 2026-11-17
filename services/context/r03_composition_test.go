package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web"
)

// A mounted router may repeat APIContexterHTTP; request context and
// cleanup must still be owned by one request scope.
func TestR03NestedAPIContexterSharesContextAndCleanup(t *testing.T) {
	parent := web.NewRouter()
	child := web.NewRouter()
	parent.AfterRouting(APIContexterHTTP())
	child.AfterRouting(APIContexterHTTP())

	var previous *APIContext
	cleanups := 0
	child.Get("/probe", func(api *APIContext) {
		if api == previous {
			t.Error("APIContext reused across HTTP requests")
		}
		previous = api
		if GetAPIContextFromRequest(api.CurrentRequest()) != api {
			t.Error("typed handler did not get request's APIContext")
		}
		store := reqctx.FromContext(api.CurrentRequest().Context())
		if store == nil {
			t.Fatal("missing request data store")
		}
		if api.GetData() == nil {
			t.Fatal("missing APIContext data")
		}
		store.AddCleanup(func() { cleanups++ })
		api.GetData()["r03"] = true
		api.Status(http.StatusNoContent)
	})
	parent.Mount("/api", child)

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		parent.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/probe", nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d: status=%d", i, rec.Code)
		}
		if cleanups != i+1 {
			t.Fatalf("request %d: cleanup=%d, want=%d", i, cleanups, i+1)
		}
	}
}

func TestR03APIContextPropagatesEnrichedRequestToTypedAction(t *testing.T) {
	router := web.NewRouter()
	router.AfterRouting(APIContexterHTTP())

	type actorKey struct{}
	router.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(context.WithValue(req.Context(), actorKey{}, "actor-42"))
			next.ServeHTTP(w, req)
		})
	})
	router.Get("/enriched", func(api *APIContext) {
		if got := api.Value(actorKey{}); got != "actor-42" {
			t.Errorf("APIContext.Value(actor)=%v", got)
		}
		if got := api.CurrentRequestContext().Value(actorKey{}); got != "actor-42" {
			t.Errorf("current request actor=%v", got)
		}
		api.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/enriched", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
}
