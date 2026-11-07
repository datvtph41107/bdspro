package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

type oauthStub struct {
	db.OAuthServerStore
	client    db.OauthClient
	inserted  int
	codeHash  []byte
	challenge string
	used      bool
}

func (s *oauthStub) GetOAuthClient(_ context.Context, id uuid.UUID) (db.OauthClient, error) {
	if id != s.client.ID {
		return db.OauthClient{}, db.ErrOAuthClientMissing
	}
	return s.client, nil
}

func (s *oauthStub) CreateOAuthAuthCode(_ context.Context, client uuid.UUID, account int64, redirect, scope, challenge string, digest []byte, _ time.Time) error {
	if client != s.client.ID || account <= 0 || redirect != s.client.RedirectUri || scope != ScopeProfile {
		return ErrInvalidRequest
	}
	s.inserted++
	s.codeHash = append([]byte(nil), digest...)
	s.challenge = challenge
	return nil
}

func (s *oauthStub) RedeemOAuthAuthCode(_ context.Context, client uuid.UUID, digest []byte, redirect, challenge string, access, refresh []byte, accessExpiry, refreshExpiry time.Time, family uuid.UUID) (db.OAuthGrant, error) {
	if s.used || client != s.client.ID || !strings.EqualFold(base64.RawURLEncoding.EncodeToString(digest), base64.RawURLEncoding.EncodeToString(s.codeHash)) || redirect != s.client.RedirectUri || challenge != s.challenge {
		return db.OAuthGrant{}, db.ErrOAuthGrantInvalid
	}
	if len(access) != 32 || len(refresh) != 32 || family == uuid.Nil || !accessExpiry.After(time.Now()) || !refreshExpiry.After(time.Now()) {
		return db.OAuthGrant{}, ErrInvalidRequest
	}
	s.used = true
	return db.OAuthGrant{AccountID: 41, Scope: ScopeProfile, RefreshExpiresAt: refreshExpiry}, nil
}

func TestAuthorizePKCEAndOneTimeCode(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	verifier := strings.Repeat("x", 43)
	digest := sha256.Sum256([]byte(verifier))
	store := &oauthStub{client: db.OauthClient{ID: id, RedirectUri: "https://client.example.test/callback"}}
	service := NewService(store)
	sessionID := uuid.New()
	req := AuthorizationRequest{
		ClientID: id, RedirectURI: store.client.RedirectUri,
		ResponseType: "code", Scope: ScopeProfile,
		CodeChallengeMethod: "S256",
		CodeChallenge:       base64.RawURLEncoding.EncodeToString(digest[:]),
		State:               "state_0123456789abcdef",
	}
	invalid := req
	invalid.CodeChallengeMethod = "plain"
	if _, err := service.Authorize(ctx, 41, sessionID, invalid); !errors.Is(err, ErrInvalidRequest) || store.inserted != 0 {
		t.Fatalf("plain PKCE accepted, err=%v, writes=%d", err, store.inserted)
	}
	invalid = req
	invalid.RedirectURI = "https://attacker.example.test/steal"
	if _, err := service.Authorize(ctx, 41, sessionID, invalid); !errors.Is(err, ErrInvalidClient) || store.inserted != 0 {
		t.Fatalf("unregistered redirect accepted: %v", err)
	}
	invalid = req
	invalid.Scope = "admin"
	if _, err := service.Authorize(ctx, 41, sessionID, invalid); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("unapproved scope accepted: %v", err)
	}
	location, err := service.Authorize(ctx, 41, sessionID, req)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	code := u.Query().Get("code")
	if u.Scheme != "https" || u.Hostname() != "client.example.test" ||
		u.Query().Get("state") != req.State || !validBearer(code) || store.inserted != 1 {
		t.Fatalf("invalid authorize redirect: %q", location)
	}
	if got := bearerHash(code); base64.RawURLEncoding.EncodeToString(got) != base64.RawURLEncoding.EncodeToString(store.codeHash) {
		t.Fatal("persisted code was not hashed")
	}
	if _, err := service.ExchangeCode(ctx, id, code, req.RedirectURI, strings.Repeat("z", 43)); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("incorrect verifier succeeded: %v", err)
	}
	result, err := service.ExchangeCode(ctx, id, code, req.RedirectURI, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if !validBearer(result.AccessToken) || !validBearer(result.RefreshToken) ||
		result.AccessToken == result.RefreshToken || result.TokenType != "Bearer" ||
		result.Scope != ScopeProfile || result.ExpiresIn <= 0 {
		t.Fatalf("invalid token response: %+v", result)
	}
	if _, err := service.ExchangeCode(ctx, id, code, req.RedirectURI, verifier); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("authorization code was accepted twice: %v", err)
	}
}

func TestOAuthRedirectPolicy(t *testing.T) {
	for _, raw := range []string{
		"https://app.example.test/callback",
		"http://localhost:8000/callback",
		"http://127.0.0.1:3000/callback",
		"https://app.example.test/callback?return=path",
	} {
		if err := ValidateRedirectURI(raw); err != nil {
			t.Errorf("valid URI rejected %q: %v", raw, err)
		}
	}
	for _, raw := range []string{
		"http://app.example.test/callback",
		"https://app.example.test/callback#fragment",
		"javascript:alert(1)",
		"https://user:pass@app.example.test/callback",
		"//attacker.example.test/",
		"http://localhost.attacker.test/callback",
		"https://app.example.test/%00",
	} {
		if ValidateRedirectURI(raw) == nil {
			t.Errorf("unsafe redirect accepted %q", raw)
		}
	}
}
