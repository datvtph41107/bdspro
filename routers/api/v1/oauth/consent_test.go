package oauth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/google/uuid"
)

func TestOAuthApproveRequiresConsentVersion(t *testing.T) {
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	router := oauthTestRouter(
		oauth_service.NewService(&noGrantStore{}),
		oauthSessionTestStore{accounts: map[uuid.UUID]int64{sessionID: 42}},
		nil, maker,
	)
	router.Post("/oauth/authorize", appctx.Authenticate(), appctx.RequireSession(), Authorize)
	jwt, _, err := maker.CreateSessionAccessToken(42, time.Minute, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"", "-1", "not-a-number"} {
		form := url.Values{"client_id": {"4df5bcd8-2cd2-496d-9ec1-cd4f99d979ce"}, "approve": {"true"}, "consent_version": {v}}
		req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+jwt)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_request") {
			t.Fatalf("consent_version=%q got %d %s", v, w.Code, w.Body.String())
		}
	}
}
