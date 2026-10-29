package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/handler"
	"github.com/datvtph41107/bdspro/internal/config"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/gin-gonic/gin"
)

func TestRouteRegistrationAndAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := token.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}

	newServer := func(google, phone bool) *Server {
		s := &Server{
			tokenMaker:          maker,
			authHandler:         handler.NewAuthHandler(nil, nil),
			organizationHandler: handler.NewOrganizationHandler(nil),
		}
		if google {
			s.googleHandler = handler.NewGoogleHandler(nil, config.Config{}, nil, nil, false)
		}
		if phone {
			s.phoneHandler = handler.NewPhoneHandler(nil, config.Config{}, nil, nil)
		}
		s.setupRouter()
		return s
	}

	base := newServer(false, false)
	want := []string{
		"GET /health", "POST /users", "POST /users/login",
		"GET /users/verify_email", "POST /users/verify_email", "POST /users/verify_email/resend", "POST /tokens/renew_access",
		"POST /tokens/revoke", "GET /users/me",
		"POST /users/password/reset/request", "GET /users/password/reset", "POST /users/password/reset",
        "POST /users/me/email/start", "POST /users/me/email/verify",
		"POST /organizations", "GET /organizations",
		"POST /organizations/:id/members", "PATCH /organizations/:id",
	}
	got := make(map[string]bool)
	for _, route := range base.router.Routes() {
		got[route.Method+" "+route.Path] = true
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
	if n := len(withOptional.router.Routes()); n != 28 {
		t.Errorf("enabled route count = %d, want 28", n)
	}
	for _, route := range []string{
		"POST /users/me/google/start", "GET /users/google/login/start",
		"GET /users/google/callback", "POST /users/me/phone/google/start",
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

	access, _, err := maker.CreateToken(42, time.Minute, token.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := maker.CreateToken(42, time.Minute, token.TokenTypeRefresh)
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
