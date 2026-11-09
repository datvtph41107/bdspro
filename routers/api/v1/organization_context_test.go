package apiv1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

func TestReqTokenRequiresVerifiedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		signed bool
		doer   bool
		status int
	}{
		{"anonymous", false, false, http.StatusUnauthorized},
		{"jwt-without-session", false, true, http.StatusUnauthorized},
		{"verified-session", true, true, http.StatusNoContent},
		{"corrupt-signed-context", true, false, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(appctx.APIContexter())
			r.Use(func(c *gin.Context) {
				ctx := appctx.GetAPIContext(c)
				ctx.IsSigned = tc.signed
				if tc.doer {
					ctx.Doer = &authcrypto.Payload{AccountID: 42}
				}
				c.Next()
			})
			reached := false
			m := web.NewRouter[*appctx.APIContext](r.Group(""), appctx.GetAPIContext)
			m.Get("/guard", reqToken(), func(ctx *appctx.APIContext) {
				reached = true
				ctx.Status(http.StatusNoContent)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/guard", nil))
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d", w.Code, tc.status)
			}
			if reached != (tc.status == http.StatusNoContent) {
				t.Fatalf("reached=%v", reached)
			}
			if tc.status == http.StatusUnauthorized {
				var response struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Code != "unauthenticated" {
					t.Fatalf("code=%s", response.Code)
				}
			}
		})
	}
}
