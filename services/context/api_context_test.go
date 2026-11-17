package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
)

func TestAPIContextIsSharedWithinRequest(t *testing.T) {
	r := web.NewRouter()
	r.AfterRouting(APIContexterHTTP())
	var first *APIContext
	r.Get("/one", func(api *APIContext) {
		first = api
		first.GetData()["check"] = true
	}, func(api *APIContext) {
		if api != first || api.GetData()["check"] != true {
			t.Fatal("APIContext was rebuilt between handlers")
		}
		api.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/one", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d", w.Code)
	}
}

type dynamicRequestKey struct{}

func TestHTTPAdapterCarriesPathAndUpdatedContext(t *testing.T) {
	router := web.NewRouter()
	router.AfterRouting(APIContexterHTTP())
	// Auth-like middleware enriches the current request after APIContexter.
	router.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(context.WithValue(req.Context(), dynamicRequestKey{}, "live"))
			next.ServeHTTP(w, req)
		})
	})
	router.Get("/orgs/{id}", func(api *APIContext) {
		id, err := api.PositivePathID("id")
		if err != nil || id != 42 {
			t.Fatalf("path ID=%d err=%v", id, err)
		}
		if got := api.Value(dynamicRequestKey{}); got != "live" {
			t.Fatalf("APIContext kept a stale request: got=%v", got)
		}
		if GetAPIContextFromRequest(api.CurrentRequest()) != api {
			t.Fatal("HTTP adapter did not reuse the same request APIContext")
		}
		api.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orgs/42", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAPIContextRegisteredOnHTTPRequest(t *testing.T) {
	router := web.NewRouter()
	router.AfterRouting(APIContexterHTTP())
	router.Get("/provider", func(api *APIContext) {
		if fromHTTP := GetAPIContextFromRequest(api.CurrentRequest()); fromHTTP != api {
			t.Fatal("HTTP handler did not resolve the same APIContext")
		}
		api.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/provider", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d", w.Code)
	}
}
