package apiv1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// The production route tree intentionally uses APIContexterHTTP, not
// the Gin-dependent APIContexter. This must hold throughout migration.
func TestH01ProductionRoutesUseNativeHTTPContext(t *testing.T) {
	r := Routes()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"status\":\"ok\"}" {
		t.Fatalf("health status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestH01DefaultChiRouterPublishesNativeHTTPPathValues(t *testing.T) {
	r := web.NewRouter()
	r.AfterRouting(appctx.APIContexterHTTP())
	r.Get("/orgs/{org}/teams/{id}", func(api *appctx.APIContext) {
		if api.PathParam("org") != "acme" {
			t.Errorf("org=%q", api.PathParam("org"))
		}
		if id, err := api.PositivePathID("id"); err != nil || id != 42 {
			t.Errorf("id=%d err=%v", id, err)
		}
		api.Status(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orgs/acme/teams/42", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=204", rec.Code)
	}
}

func TestH01NativeHTTPContextAcrossMountedRouter(t *testing.T) {
	parent, child := web.NewRouter(), web.NewRouter()
	child.AfterRouting(appctx.APIContexterHTTP())
	child.Get("/teams/{team}", func(api *appctx.APIContext) {
		if api.PathParam("org") != "acme" || api.PathParam("team") != "backend" {
			t.Errorf("route params org=%q team=%q", api.PathParam("org"), api.PathParam("team"))
		}
		api.JSON(http.StatusOK, map[string]bool{"ok": true})
	})
	parent.Mount("/orgs/{org}", child)
	rec := httptest.NewRecorder()
	parent.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orgs/acme/teams/backend", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"ok\":true}" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}
