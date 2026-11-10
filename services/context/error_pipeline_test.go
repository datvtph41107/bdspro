package context

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/gin-gonic/gin"
	"github.com/datvtph41107/bdspro/modules/web"
)

// Trace model error -> util -> API context -> Gin adapter -> HTTP response.
func TestAPIErrorPipelineRetainsPublicContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "private postgres password=secret"
	cases := []struct {
		name, code, message string
		status              int
		invoke              func(*APIContext)
	}{
		{
			name: "typed API not found", code: "not_found", message: "not found", status: http.StatusNotFound,
			invoke: func(c *APIContext) { c.APIErrorNotFound() },
		},
		{
			name: "typed public failure", code: "owner_required", message: "owner required", status: http.StatusForbidden,
			invoke: func(c *APIContext) { c.APIError(http.StatusForbidden, "owner_required", "owner required") },
		},
		{
			name: "public domain error hides cause", code: "organization.membership_exists",
			message: "membership already exists", status: http.StatusConflict,
			invoke: func(c *APIContext) {
				c.APIErrorAuto(util.WithCause(organization_model.ErrMembershipExists, errors.New(secret)))
			},
		},
		{
			name: "unexpected error is masked", code: "internal_error",
			message: "internal server error", status: http.StatusInternalServerError,
			invoke: func(c *APIContext) { c.APIErrorAuto(errors.New(secret)) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := web.NewRouter()
			router.AfterRouting(APIContexter())
			continued := false
			router.Get("/", func(c *APIContext) {
				tc.invoke(c)
				if !c.IsAborted() {
					t.Error("error response did not abort remaining handlers")
				}
				source := c.Base.source.(ginRequestSource).ctx
				if got := source.GetString(ErrorCodeKey); got != tc.code {
					t.Errorf("monitor code=%q want=%q", got, tc.code)
				}
			}, func(*APIContext) { continued = true })
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			if continued || w.Code != tc.status {
				t.Fatalf("continued=%v status=%d want=%d", continued, w.Code, tc.status)
			}
			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("content type=%q", ct)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 2 || body["error"] != tc.message || body["code"] != tc.code {
				t.Fatalf("error response mismatch: %#v", body)
			}
			if strings.Contains(w.Body.String(), secret) {
				t.Fatal("private error leaked to client")
			}
		})
	}
}
