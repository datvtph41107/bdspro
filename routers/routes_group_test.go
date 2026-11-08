package routers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	"github.com/gin-gonic/gin"
)

// Route grouping must never allow a protected endpoint to skip its parent
// authentication middleware.
func TestOrganizationGroupInheritsAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{
		tokenMaker:          maker,
		authHandler:         user.NewAuthHandler(nil, nil, nil),
		organizationHandler: org.NewHandler(nil),
	}
	router := gin.New()
	apiv1.Routes(router, apiv1.Dependencies{Auth: server.authHandler, Organization: server.organizationHandler, TokenMaker: server.tokenMaker, Store: server.store})

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/organizations"},
		{http.MethodPost, "/organizations"},
		{http.MethodGet, "/organizations/1"},
		{http.MethodPatch, "/organizations/1"},
		{http.MethodGet, "/organizations/1/members"},
		{http.MethodPost, "/organizations/1/members"},
		{http.MethodDelete, "/organizations/1/members/2"},
		{http.MethodPut, "/organizations/1/owners/2"},
		{http.MethodDelete, "/organizations/1/owners/2"},
		{http.MethodPost, "/organizations/1/owners/transfer"},
		{http.MethodGet, "/organizations/1/teams"},
		{http.MethodPost, "/organizations/1/teams"},
		{http.MethodGet, "/teams/1"},
		{http.MethodPatch, "/teams/1"},
		{http.MethodPost, "/teams/1/members"},
		{http.MethodDelete, "/teams/1/members/2"},
		{http.MethodGet, "/teams/1/members"},
		{http.MethodPost, "/teams/1/invites"},
		{http.MethodDelete, "/teams/1/invites/3e7783e8-a93b-4875-ab34-4f95bc150000"},
		{http.MethodPost, "/teams/invites/accept"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(route.method, route.path, nil))
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status %d, want 401", recorder.Code)
			}
			if got := recorder.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Fatalf("missing Bearer challenge: %q", got)
			}
		})
	}
}
