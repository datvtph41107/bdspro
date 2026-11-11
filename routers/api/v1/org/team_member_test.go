package org

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

func TestTeamMemberActionsRejectInvalidInputsBeforeDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		method, path, pattern string
		action                func(*appctx.APIContext)
	}{
		{http.MethodGet, "/teams/invalid/members", "/teams/:team_id/members", ListTeamMembers},
		{http.MethodGet, "/teams/1/members?page=invalid", "/teams/:team_id/members", ListTeamMembers},
		{http.MethodGet, "/teams/1/members?limit=-1", "/teams/:team_id/members", ListTeamMembers},
		{http.MethodPost, "/teams/invalid/members", "/teams/:team_id/members", AddTeamMember},
		{http.MethodDelete, "/teams/1/members/invalid", "/teams/:team_id/members/:person_id", RemoveTeamMember},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			r := web.NewRouter()
			r.AfterRouting(appctx.APIContexter())
			r.Methods(tc.method, tc.pattern, tc.action)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=400: %s", w.Code, w.Body.String())
			}
		})
	}
}
