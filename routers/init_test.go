package routers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
)

// Route compilation must not require PostgreSQL, Redis, JWT or the Manager.
func TestPrepareWebRoutesWithoutRuntimeResources(t *testing.T) {
	before := setting.RunMode
	t.Cleanup(func() { setting.RunMode = before })
	setting.RunMode = "test"
	routes, err := PrepareWebRoutes()
	if err != nil {
		t.Fatalf("prepare HTTP routes: %v", err)
	}
	recorder := httptest.NewRecorder()
	routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "ok") {
		t.Fatalf("health result: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestPrepareWebRoutesRejectsInvalidTrustedProxyBeforeResources(t *testing.T) {
	beforeRunMode, beforeServer := setting.RunMode, setting.Server
	t.Cleanup(func() {
		setting.RunMode, setting.Server = beforeRunMode, beforeServer
	})
	setting.RunMode = "test"
	setting.Server.TrustedProxies = []string{"not-a-proxy"}
	_, err := PrepareWebRoutes()
	if err == nil || !strings.Contains(err.Error(), "trusted proxy") {
		t.Fatalf("expected a route preflight error, got %v", err)
	}
}

func TestRouterRejectsProgrammingErrorDuringRegistration(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("unsupported handler signature must panic before resource initialization")
		}
	}()
	routes := web.NewRouter()
	routes.Get("/invalid", func() string { return "invalid" })
}

func TestNormalHandlerRejectsMissingPreparedRoutes(t *testing.T) {
	if _, err := NormalHandler(nil); err == nil {
		t.Fatal("missing static HTTP routes accepted")
	}
}
