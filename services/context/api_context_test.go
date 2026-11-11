package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/gin-gonic/gin"
)

func TestAPIContextIsSharedWithinRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := web.NewRouter()
	r.AfterRouting(APIContexter())
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

func TestGinAdapterCarriesPathAndUpdatedContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.Get("/orgs/:id", func(api *APIContext) {
		c := api.Base.source.(ginRequestSource).ctx
		id, err := api.PositivePathID("id")
		if err != nil || id != 42 {
			t.Fatalf("path ID=%d err=%v", id, err)
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), dynamicRequestKey{}, "live"))
		if got := api.Value(dynamicRequestKey{}); got != "live" {
			t.Fatalf("APIContext kept a stale request: got=%v", got)
		}
		if GetAPIContext(c) != api {
			t.Fatal("Gin adapter did not reuse the same request APIContext")
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
	gin.SetMode(gin.TestMode)
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.Get("/provider", func(api *APIContext) {
		if fromHTTP := GetAPIContextFromRequest(api.CurrentRequest()); fromHTTP != api {
			t.Fatal("net/http and Gin did not resolve the same APIContext")
		}
		api.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/provider", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d", w.Code)
	}
}
