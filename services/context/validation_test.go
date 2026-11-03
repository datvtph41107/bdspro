package context

import (
	"github.com/datvtph41107/bdspro/modules/reqctx"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBindJSONContract(t *testing.T) {
	for _, tc := range []struct {
		body, code string
		status     int
	}{
		{"", "request_body_required", 400},
		{"{", "malformed_json", 400},
		{"{\"name\": 8}", "validation_failed", 400},
		{"{} {}", "invalid_request_body", 400},
	} {
		t.Run(tc.code+tc.body, func(t *testing.T) {
			w := httptest.NewRecorder()
			reqctx.RequestContextHandler()(APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var target struct {
					Name string `json:"name"`
				}
				if GetAPIContext(r).BindJSON(&target) {
					t.Fatal("invalid input accepted")
				}
			}))).ServeHTTP(w, httptest.NewRequest("POST", "/", strings.NewReader(tc.body)))
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

