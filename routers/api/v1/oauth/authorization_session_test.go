package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type oauthAuthorizationSessionReader struct {
	db.Store
	session db.Session
	err     error
}

func (s *oauthAuthorizationSessionReader) GetSession(_ context.Context, _ uuid.UUID) (db.Session, error) {
	if s.err != nil {
		return db.Session{}, s.err
	}
	return s.session, nil
}

func TestOAuthAuthorizeBearerMustHaveLiveSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sessionID := uuid.New()
	store := &oauthAuthorizationSessionReader{session: db.Session{
		ID: sessionID, AccountID: 42, ExpiresAt: time.Now().Add(time.Hour),
	}}
	clientID := uuid.New()
	oauthStore := &consentProofHTTPStore{client: db.OauthClient{
		ID: clientID, RedirectUri: "https://client.example.test/callback",
	}}
	proofs, err := oauth_service.NewConsentProofs(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(oauth_service.NewService(oauthStore), store, proofs)
	router := gin.New()
	router.GET("/oauth/authorize", handler.Authenticate(maker), handler.AuthorizePreview)
	router.POST("/oauth/authorize", handler.Authenticate(maker), handler.Authorize)
	// This endpoint deliberately retains the ordinary stateless access JWT policy.
	router.GET("/api/account", appctx.Authenticate(maker), func(c *gin.Context) { c.Status(http.StatusOK) })
	challenge := sha256.Sum256([]byte(strings.Repeat("x", 43)))
	query := url.Values{
		"client_id": {clientID.String()}, "redirect_uri": {oauthStore.client.RedirectUri},
		"response_type": {"code"}, "scope": {"profile:read"}, "state": {"0123456789abcdef"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	get := func(jwt string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query.Encode(), nil)
		req.Header.Set("Authorization", "Bearer "+jwt)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	legacy, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if got := get(legacy); got.Code != http.StatusUnauthorized {
		t.Fatalf("legacy unbound JWT issued OAuth consent preview: %d", got.Code)
	}
	apiReq := httptest.NewRequest(http.MethodGet, "/api/account", nil)
	apiReq.Header.Set("Authorization", "Bearer "+legacy)
	apiRec := httptest.NewRecorder()
	router.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusOK {
		t.Fatalf("legacy API JWT behavior unexpectedly changed: %d", apiRec.Code)
	}
	bound, _, err := maker.CreateSessionAccessToken(42, time.Minute, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	preview := get(bound)
	if preview.Code != http.StatusOK {
		t.Fatalf("live session preview=%d %s", preview.Code, preview.Body.String())
	}
	var consent struct {
		Proof   string `json:"consent_proof"`
		Version int64  `json:"consent_version"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &consent); err != nil || consent.Proof == "" {
		t.Fatalf("invalid preview: %s %v", preview.Body.String(), err)
	}
	approve := url.Values{}
	for k, v := range query {
		approve[k] = append([]string(nil), v...)
	}
	approve.Set("approve", "true")
	approve.Set("consent_proof", consent.Proof)
	approve.Set("consent_version", strconv.FormatInt(consent.Version, 10))
	post := func(jwt string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(approve.Encode()))
		req.Header.Set("Authorization", "Bearer "+jwt)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	store.session.IsBlocked = true
	if got := get(bound); got.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session accepted GET preview=%d", got.Code)
	}
	if got := post(bound); got.Code != http.StatusUnauthorized || oauthStore.approvals != 0 {
		t.Fatalf("revoked session issued consent code=%d approvals=%d", got.Code, oauthStore.approvals)
	}
	store.session.IsBlocked = false
	store.err = errors.New("postgres unavailable")
	if got := get(bound); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("DB outage passed as auth failure: %d", got.Code)
	}
	store.err = nil
	if got := post(bound); got.Code != http.StatusFound || oauthStore.approvals != 1 {
		t.Fatalf("valid session could not authorize=%d approvals=%d", got.Code, oauthStore.approvals)
	}
	store.err = pgx.ErrNoRows
	if got := get(bound); got.Code != http.StatusUnauthorized {
		t.Fatalf("deleted session was not rejected: %d", got.Code)
	}
}
