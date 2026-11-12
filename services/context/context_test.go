package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web"
)

func TestAPIContexterSingleContext(t *testing.T) {
	requestCtx, done := reqctx.NewRequestContext(context.Background())
	defer done()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestCtx)
	out := httptest.NewRecorder()
	var first *APIContext
	handler := APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		first = GetAPIContext(req)
		if first != GetAPIContext(req) {
			t.Fatal("APIContext changed")
		}
		first.Data["same"] = true
		if reqctx.FromContext(req.Context()).GetData()["same"] != true {
			t.Fatal("second datastore")
		}
		first.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}))
	handler.ServeHTTP(out, req)
	if first == nil || first.WrittenStatus() != http.StatusOK || out.Code != http.StatusOK {
		t.Fatalf("unexpected APIContext or status: %d", out.Code)
	}
}

func TestTypedAPIErrorStopsEndpoint(t *testing.T) {
	r := web.NewRouter()
	r.AfterRouting(APIContexter())
	reached := false
	r.Get("/test",
		func(ctx *APIContext) {
			ctx.APIError(http.StatusUnauthorized, "token required")
		},
		func(ctx *APIContext) {
			reached = true
			ctx.JSON(http.StatusOK, "unexpected")
		},
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))
	if reached {
		t.Fatal("typed endpoint executed after API error")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=401", w.Code)
	}
}
