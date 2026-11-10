package org

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

// bindTestForm exercises the same modules/web form boundary as api/v1.bind.
func bindTestForm[T any](ctx *appctx.APIContext) {
	form, err := middleware.BindJSON[T](ctx)
	if err != nil {
		ctx.Validation(err, form)
		return
	}
	web.SetForm(ctx, form)
}

func TestTeamPackageActionsRejectMalformedInputsWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name, method, path, pattern, body string
		bind                              func(*appctx.APIContext)
		action                            func(*appctx.APIContext)
	}{
		{
			name:   "create requires name",
			method: http.MethodPost, path: "/orgs/1/teams", pattern: "/orgs/:id/teams", body: "{}",
			bind: bindTestForm[api.CreateTeamOption], action: CreateTeam,
		},
		{
			name:   "edit requires name",
			method: http.MethodPatch, path: "/teams/1", pattern: "/teams/:team_id", body: "{}",
			bind: bindTestForm[api.RenameTeamOption], action: EditTeam,
		},
		{
			name:   "edit rejects invalid team ID",
			method: http.MethodPatch, path: "/teams/invalid", pattern: "/teams/:team_id", body: `{"name":"Renamed"}`,
			bind: bindTestForm[api.RenameTeamOption], action: EditTeam,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(appctx.APIContexter())
			m := web.NewRouter(router)
			m.Methods(tc.method, tc.pattern, tc.bind, tc.action)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=400: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestListTeamsActionRejectsInvalidPaginationWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{
		"/orgs/1/teams?page=invalid",
		"/orgs/1/teams?limit=0",
		"/orgs/1/teams?page=-1",
		"/orgs/1/teams?page=999999999999999999999999999",
	} {
		t.Run(path, func(t *testing.T) {
			router := gin.New()
			router.GET("/orgs/:id/teams", appctx.Wrap(ListTeams))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=400: %s", response.Code, response.Body.String())
			}
		})
	}
}
