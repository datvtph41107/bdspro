package apiv1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// A signed JWT is not an authorized request without a matching live session.
// The actor in context.Context is the single authority source for model calls.
func TestReqTokenRequiresVerifiedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name            string
		signed          bool
		doerID          int64
		actorID         int64
		sessionMismatch bool
		status          int
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

func TestOrgAssignmentRejectsInvalidIDForTeamListBeforeDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(appctx.APIContexter())
	r.Use(func(c *gin.Context) {
		api := appctx.GetAPIContext(c)
		sessionID := uuid.MustParse("e27803ce-3e4f-42d5-94d6-7693d30b825d")
		api.IsSigned = true
		api.Doer = &authcrypto.Payload{AccountID: 42, SessionID: sessionID}
		c.Request = c.Request.WithContext(auth_model.WithActor(c.Request.Context(), auth_model.Actor{
			AccountID: 42, SessionID: sessionID,
		}))
		c.Next()
	})
	reached := false
	m := web.NewRouter[*appctx.APIContext](r.Group(""), appctx.GetAPIContext)
	m.Group("/orgs/:id", func() {
		m.Get("/teams", func(*appctx.APIContext) { reached = true })
	}, orgAssignment(nil))
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/orgs/invalid/teams", nil))
	if response.Code != http.StatusBadRequest || reached {
		t.Fatalf("bad organization ID passed assignment: status=%d reached=%v body=%s",
			response.Code, reached, response.Body.String())
	}
}

type providerIngressKey struct{}

func TestAfterRoutingHTTPPreAndTypedAPIContextShareRequestLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	m := web.NewRouter[*appctx.APIContext](engine.Group(""), appctx.GetAPIContext)
	m.AfterRouting(
		appctx.APIContexter(),
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), providerIngressKey{}, "from-http")))
			})
		},
	)
	var order []string
	var first *appctx.APIContext
	m.Group("/teams", func() {
		m.Post("", web_types.PreMiddlewareProvider(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, "pre-route")
				next.ServeHTTP(w, req)
			})
		}), func(ctx *appctx.APIContext) {
			order = append(order, "action")
			first = ctx
			if ctx.Value(providerIngressKey{}) != "from-http" || appctx.GetAPIContextFromRequest(ctx.CurrentRequest()) != ctx {
				t.Error("HTTP request context and typed APIContext diverged")
			}
			ctx.JSON(http.StatusCreated, map[string]bool{"ok": true})
		}, func(*appctx.APIContext) {
			t.Error("action after committed response executed")
		})
	}, func(ctx *appctx.APIContext) {
		order = append(order, "group")
		if first != nil {
			t.Error("typed endpoint context existed before guard")
		}
	})
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/teams", nil))
	if response.Code != http.StatusCreated || first == nil ||
		!reflect.DeepEqual(order, []string{"pre-route", "group", "action"}) {
		t.Fatalf("status=%d order=%v APIContext=%v", response.Code, order, first != nil)
	}
}
