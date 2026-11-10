package org

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

// Package actions use the same typed binder as production routes.
func TestTeamInvitePackageActionsRejectMalformedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path, pattern, body string
		bind                              func(*appctx.APIContext)
		action                            func(*appctx.APIContext)
	}{
		{
			name: "missing email", method: http.MethodPost,
			path: "/teams/1/invites", pattern: "/teams/:team_id/invites", body: "{}",
			bind: bindTestForm[api.CreateTeamInviteOption], action: CreateInvite(""),
		},
		{
			name: "invalid email", method: http.MethodPost,
			path: "/teams/1/invites", pattern: "/teams/:team_id/invites", body: `{"email":"bad"}`,
			bind: bindTestForm[api.CreateTeamInviteOption], action: CreateInvite(""),
		},
		{
			name: "invalid team", method: http.MethodPost,
			path: "/teams/invalid/invites", pattern: "/teams/:team_id/invites", body: `{"email":"member@example.test"}`,
			bind: bindTestForm[api.CreateTeamInviteOption], action: CreateInvite(""),
		},
		{
			name: "missing token", method: http.MethodPost,
			path: "/teams/invites/accept", pattern: "/teams/invites/accept", body: "{}",
			bind: bindTestForm[api.AcceptTeamInviteOption], action: AcceptInvite,
		},
		{
			name: "invalid token length", method: http.MethodPost,
			path: "/teams/invites/accept", pattern: "/teams/invites/accept", body: `{"token":"wrong"}`,
			bind: bindTestForm[api.AcceptTeamInviteOption], action: AcceptInvite,
		},
		{
			name: "invalid invite ID", method: http.MethodDelete,
			path: "/teams/1/invites/wrong", pattern: "/teams/:team_id/invites/:invite_id",
			action: RevokeInvite,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(appctx.APIContexter())
			m := web.NewRouter(router)
			if tc.bind != nil {
				m.Methods(tc.method, tc.pattern, tc.bind, tc.action)
			} else {
				m.Methods(tc.method, tc.pattern, tc.action)
			}
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
