package routers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The registered route reads a real account through AuthService, while this
// test remains independent from PostgreSQL.
type routeAccountStore struct {
	db.Store
	sessionID uuid.UUID
}

func (s routeAccountStore) GetSession(_ context.Context, id uuid.UUID) (db.Session, error) {
	if id != s.sessionID {
		return db.Session{}, pgx.ErrNoRows
	}
	return db.Session{ID: id, AccountID: 42, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (routeAccountStore) GetAccountProfile(_ context.Context, id int64) (db.GetAccountProfileRow, error) {
	return db.GetAccountProfileRow{AccountID: id, PersonID: id + 1, Email: "route@example.test",
		CreatedAt:       time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC),
		IsEmailVerified: true, Locale: "vi", Timezone: "UTC", Currency: "VND"}, nil
}

func (routeAccountStore) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	return db.Account{ID: id, Email: "route@example.test", CreatedAt: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC), IsEmailVerified: true}, nil
}

func TestRouteRegistrationAndAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}

	sid := uuid.New()
	newServer := func(oauth2, phone bool) *Server {
		store := routeAccountStore{sessionID: sid}
		s := &Server{
			tokenMaker:           maker,
			store:                store,
			authHandler:          user.NewAuthHandler(auth_service.NewAuthService(store, nil, setting.Config{}, nil), nil, nil),
			organizationHandler:  org.NewHandler(nil),
			authorizationHandler: oauth.NewHandler(nil, nil),
		}
		if oauth2 {
			s.oauth2Handler = user.NewOAuth2Handler(nil, nil, false)
		}
		if phone {
			s.phoneEnabled = true
		}
		s.router = NormalRoutes(s)
		return s
	}

	base := newServer(false, false)
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
	got := make(map[string]bool)
	for _, route := range base.router.Routes() {
		got[route.Method+" "+route.Path] = true
		if strings.HasPrefix(route.Path, "/organizations") { t.Fatalf("legacy /organizations route still registered: %s %s", route.Method, route.Path) }
	}
	if len(got) != len(want) {
		t.Fatalf("base route count = %d, want %d: %v", len(got), len(want), got)
	}
	for _, route := range want {
		if !got[route] {
			t.Errorf("missing route %s", route)
		}
	}

	withOptional := newServer(true, true)
	if n := len(withOptional.router.Routes()); n != 61 {
		t.Errorf("enabled route count = %d, want 61", n)
	}
	for _, route := range []string{
		"POST /users/me/oauth2/start", "GET /users/oauth2/login/start",
		"GET /users/oauth2/callback", "POST /users/me/phone/oauth2/start",
		"POST /users/me/phone/start",
		"POST /users/me/phone/verify",
		"POST /users/phone/register/start", "POST /users/phone/register/verify",
		"POST /users/phone/login/start", "POST /users/phone/login/verify",
	} {
		found := false
		for _, current := range withOptional.router.Routes() {
			if current.Method+" "+current.Path == route {
				found = true
			}
		}
		if !found {
			t.Errorf("missing optional route %s", route)
		}
	}

	for _, removed := range []string{"POST /users/phone/password/login"} {
		for _, current := range withOptional.router.Routes() {
			if current.Method+" "+current.Path == removed {
				t.Errorf("unsupported phone/password route exposed: %s", removed)
			}
		}
	}

	access, _, err := maker.CreateSessionAccessToken(42, time.Minute, sid)
	if err != nil {
		t.Fatal(err)
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
		{"access", "Bearer " + access, http.StatusOK},
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
