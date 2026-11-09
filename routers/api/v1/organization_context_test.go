package apiv1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// A signed JWT is not an authorized request without a matching live session.
// The actor in context.Context is the single authority source for model calls.
func TestReqTokenRequiresVerifiedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name string
		signed bool
		doerID int64
		actorID int64
		sessionMismatch bool
		status int
	}{
		{"anonymous", false, 0, 0, false, http.StatusUnauthorized},
		{"jwt-without-session", false, 42, 0, false, http.StatusUnauthorized},
		{"signed-flag-without-actor", true, 42, 0, false, http.StatusUnauthorized},
		{"actor-without-signed-flag", false, 42, 42, false, http.StatusUnauthorized},
		{"verified-session", true, 42, 42, false, http.StatusNoContent},
		{"mismatched-account", true, 42, 17, false, http.StatusUnauthorized},
		{"mismatched-session", true, 42, 42, true, http.StatusUnauthorized},
		{"corrupt-signed-context", true, 0, 42, false, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(appctx.APIContexter())
			r.Use(func(c *gin.Context) {
				ctx := appctx.GetAPIContext(c)
				ctx.IsSigned = tc.signed
				sessionID := uuid.MustParse("e27803ce-3e4f-42d5-94d6-7693d30b825d")
				if tc.doerID > 0 {
					ctx.Doer = &authcrypto.Payload{AccountID: tc.doerID, SessionID: sessionID}
				}
				if tc.actorID > 0 {
					actorSessionID := sessionID
					if tc.sessionMismatch {
						actorSessionID = uuid.MustParse("3bd93bd9-80f8-426a-874a-3f6a4f472009")
					}
					c.Request = c.Request.WithContext(auth_model.WithActor(c.Request.Context(), auth_model.Actor{
						AccountID: tc.actorID,
						SessionID: actorSessionID,
					}))
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

func TestOrgAssignmentRejectsMismatchedAuthorityBeforeDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(appctx.APIContexter())
	r.Use(func(c *gin.Context) {
		api := appctx.GetAPIContext(c)
		api.IsSigned = true
		api.Doer = &authcrypto.Payload{
			AccountID: 42,
			SessionID: uuid.MustParse("e27803ce-3e4f-42d5-94d6-7693d30b825d"),
		}
		c.Next()
	})
	reached := false
	m := web.NewRouter[*appctx.APIContext](r.Group(""), appctx.GetAPIContext)
	m.Group("/orgs/:id", func() {
		m.Get("", func(*appctx.APIContext) { reached = true })
	}, orgAssignment(nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orgs/42", nil))
	if w.Code != http.StatusUnauthorized || reached {
		t.Fatalf("status=%d reached=%v body=%s", w.Code, reached, w.Body.String())
	}
}
