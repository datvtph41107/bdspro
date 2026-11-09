package org

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

func TestTeamMemberActionsRejectInvalidInputsBeforeDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		method, path, pattern string
		action appctx.APIHandler
	}{
		{http.MethodGet, "/teams/invalid/members", "/teams/:team_id/members", ListTeamMembers},
		{http.MethodGet, "/teams/1/members?page=invalid", "/teams/:team_id/members", ListTeamMembers},
		{http.MethodGet, "/teams/1/members?limit=-1", "/teams/:team_id/members", ListTeamMembers},
		{http.MethodPost, "/teams/invalid/members", "/teams/:team_id/members", AddTeamMember},
		{http.MethodDelete, "/teams/1/members/invalid", "/teams/:team_id/members/:person_id", RemoveTeamMember},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			r := gin.New()
			r.Use(appctx.APIContexter())
			r.Handle(tc.method, tc.pattern, appctx.Wrap(tc.action))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=400: %s", w.Code, w.Body.String())
			}
		})
	}
}
