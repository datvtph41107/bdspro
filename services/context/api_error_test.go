package context

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

func TestAPIErrorPreservesPublicEnvelope(t *testing.T) {
	scoped, done := reqctx.NewRequestContext(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	defer done()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(scoped)
	out := httptest.NewRecorder()
	APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		GetAPIContext(req).APIError(http.StatusUnauthorized, "authentication required")
	})).ServeHTTP(out, req)
	if out.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", out.Code)
	}
	var response APIError
	if err := json.Unmarshal(out.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "authentication required" || response.Code != "unauthenticated" {
		t.Fatalf("public envelope changed: %+v", response)
	}
}
