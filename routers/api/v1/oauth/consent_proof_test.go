package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
)

type consentProofHTTPStore struct {
	db.OAuthServerStore
	client    db.OauthClient
	version   int64
	approvals int
}

func (s *consentProofHTTPStore) GetOAuthClient(_ context.Context, id uuid.UUID) (db.OauthClient, error) {
	if id != s.client.ID {
		return db.OauthClient{}, db.ErrOAuthClientMissing
	}
	return s.client, nil
}
func (s *consentProofHTTPStore) GetOAuthConsentVersion(_ context.Context, _ int64, _ uuid.UUID) (int64, error) {
	return s.version, nil
}
func (s *consentProofHTTPStore) ApproveOAuthAuthCode(_ context.Context, _ uuid.UUID, _ int64,
	_, _, _ string, _ []byte, _ time.Time, v int64, sessionID uuid.UUID) error {
	if sessionID == uuid.Nil {
		return db.ErrOAuthSessionInactive
	}
	if v != s.version {
		return db.ErrOAuthConsentStale
	}
	s.version++
	s.approvals++
	return nil
}

func TestOAuthConsentProofBindsHTTPPreviewAndApproval(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()
	store := &consentProofHTTPStore{client: db.OauthClient{ID: id, RedirectUri: "https://client.example.test/callback"}}
	signer, err := oauth_service.NewConsentProofs(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(oauth_service.NewService(store), nil, signer)
	r := gin.New()
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	grp := r.Group("", appctx.Authenticate(maker))
	grp.GET("/oauth/authorize", handler.AuthorizePreview)
	grp.POST("/oauth/authorize", handler.Authorize)
	bearer, _, err := maker.CreateSessionAccessToken(42, time.Minute, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	wrongBearer, _, err := maker.CreateSessionAccessToken(43, time.Minute, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	pkce := sha256.Sum256([]byte(strings.Repeat("x", 43)))
	form := url.Values{
		"client_id": {id.String()}, "redirect_uri": {store.client.RedirectUri},
		"response_type": {"code"}, "scope": {"profile:read"},
		"state": {"0123456789abcdef"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(pkce[:])},
		"code_challenge_method": {"S256"},
	}
	preview := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+form.Encode(), nil)
	preview.Header.Set("Authorization", "Bearer "+bearer)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, preview)
	if w.Code != http.StatusOK {
		t.Fatalf("preview=%d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("unsafe preview headers: %v", w.Header())
	}
	var payload struct {
		Proof   string `json:"consent_proof"`
		Version int64  `json:"consent_version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Proof == "" {
		t.Fatal("missing signed consent proof")
	}
	form.Set("approve", "true")
	form.Set("consent_version", strconv.FormatInt(payload.Version, 10))
	form.Set("consent_proof", payload.Proof)
	post := func(v url.Values, auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(v.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+auth)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	for name, mutate := range map[string]func(url.Values){
		"missing proof": func(v url.Values) { v.Del("consent_proof") },
		"changed state": func(v url.Values) { v.Set("state", "changed_state_0123456789") },
		"changed challenge": func(v url.Values) {
			v.Set("code_challenge", base64.RawURLEncoding.EncodeToString(sha256.New().Sum(nil)))
		},
		"changed version": func(v url.Values) { v.Set("consent_version", "1") },
	} {
		t.Run(name, func(t *testing.T) {
			v := url.Values{}
			for k, vals := range form {
				v[k] = append([]string(nil), vals...)
			}
			mutate(v)
			rec := post(v, bearer)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
			}
			if store.approvals != 0 {
				t.Fatal("invalid approval mutated consent")
			}
		})
	}
	if rec := post(form, wrongBearer); rec.Code != http.StatusBadRequest {
		t.Fatalf("cross-account proof accepted: %d", rec.Code)
	}
	if rec := post(form, bearer); rec.Code != http.StatusFound {
		t.Fatalf("valid approval rejected %d: %s", rec.Code, rec.Body.String())
	}
	if store.approvals != 1 {
		t.Fatalf("approval count=%d", store.approvals)
	}
	if rec := post(form, bearer); rec.Code != http.StatusBadRequest {
		t.Fatalf("replayed approval accepted: %d", rec.Code)
	}
}
