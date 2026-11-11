package routers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/gin-gonic/gin"
)

func TestRouteRegistrationAndAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}

	routes := NormalRoutes()
	base := &serverFixture{
		tokenMaker: maker, router: routes.GinEngine(),
	}
	// Route registration needs no database. The successful authenticated request
	// path is covered by the real-PostgreSQL G12 session HTTP acceptance tests.
	base.httpHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := authcrypto.WithMaker(req.Context(), maker)
		routes.ServeHTTP(w, req.WithContext(ctx))
	})

	want := []string{
		"GET /oauth/authorize", "POST /oauth/authorize",
		"POST /oauth/token", "POST /oauth/revoke", "GET /oauth/userinfo",
		"GET /oauth/clients", "POST /oauth/clients", "DELETE /oauth/clients/:id",
		"GET /users/me/oauth/authorizations", "DELETE /users/me/oauth/authorizations/:client_id",
		"GET /health", "POST /users", "POST /users/login",
		"GET /users/verify_email", "POST /users/verify_email", "POST /users/verify_email/resend", "POST /tokens/renew_access",
		"POST /tokens/revoke", "GET /users/me",
		"PATCH /users/me/profile", "GET /users/me/settings", "PATCH /users/me/settings",
		"POST /users/password/reset/request", "GET /users/password/reset", "POST /users/password/reset",
		"POST /users/me/email/start", "POST /users/me/email/verify",
		"GET /users/me/connections", "GET /users/me/sessions", "DELETE /users/me/sessions/:id", "POST /users/me/sessions/revoke_all",
		"POST /orgs", "GET /orgs",
		"POST /orgs/:id/members", "DELETE /orgs/:id/members/:person_id", "PATCH /orgs/:id",
		"PUT /orgs/:id/owners/:person_id", "DELETE /orgs/:id/owners/:person_id",
		"POST /orgs/:id/owners/transfer",
		"GET /orgs/:id", "GET /orgs/:id/members",
		"GET /orgs/:id/teams", "POST /orgs/:id/teams",
		"GET /teams/:team_id", "PATCH /teams/:team_id",
		"POST /teams/:team_id/members", "DELETE /teams/:team_id/members/:person_id",
		"GET /teams/:team_id/members", "POST /teams/:team_id/invites",
		"DELETE /teams/:team_id/invites/:invite_id", "POST /teams/invites/accept",
	}
	// Optional features have stable route registration. Their package guards
	// decide availability per request, not per process-wide router state.
	want = append(want,
		"POST /users/me/oauth2/start",
		"GET /users/oauth2/login/start",
		"GET /users/oauth2/callback",
		"POST /users/me/phone/oauth2/start",
		"POST /users/me/phone/start",
		"POST /users/me/phone/verify",
		"POST /users/phone/register/start",
		"POST /users/phone/register/verify",
		"POST /users/phone/login/start",
		"POST /users/phone/login/verify",
		"GET /oauth/login",
		"POST /oauth/login",
		"POST /oauth/logout",
		"GET /oauth/login/oidc/start",
	)
	got := make(map[string]bool)
	for _, route := range base.router.Routes() {
		got[route.Method+" "+route.Path] = true
		if strings.HasPrefix(route.Path, "/organizations") {
			t.Fatalf("legacy /organizations route still registered: %s %s", route.Method, route.Path)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("base route count = %d, want %d: %v", len(got), len(want), got)
	}
	for _, route := range want {
		if !got[route] {
			t.Errorf("missing route %s", route)
		}
	}

	// A second, independently constructed router has the same route inventory.
	// This proves route declaration no longer depends on InitRuntime flags.
	again := NormalRoutes().GinEngine()
	if len(again.Routes()) != len(base.router.Routes()) {
		t.Fatalf("route inventory depends on process-level initialization: first=%d second=%d",
			len(base.router.Routes()), len(again.Routes()))
	}
	for _, current := range again.Routes() {
		if !got[current.Method+" "+current.Path] {
			t.Errorf("route missing in first registration: %s %s", current.Method, current.Path)
		}
	}

	for _, removed := range []string{"POST /users/phone/password/login"} {
		for _, current := range base.router.Routes() {
			if current.Method+" "+current.Path == removed {
				t.Errorf("unsupported phone/password route exposed: %s", removed)
			}
		}
	}

	refresh, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeRefresh)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		header string
		status int
	}{
		{"missing", "", http.StatusUnauthorized},
		{"refresh", "Bearer " + refresh, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/users/me", nil)
			if tc.header != "" {
				r.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			base.Handler().ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d; body = %s", w.Code, tc.status, w.Body.String())
			}
		})
	}
}
