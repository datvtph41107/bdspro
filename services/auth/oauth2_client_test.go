package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/go-redis/redis/v8"
)

func TestOAuth2ClientTokenExchange(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("wrong method: %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.Form.Get("grant_type") != "authorization_code" ||
			r.Form.Get("code_verifier") != "verifier" {
			t.Errorf("missing oauth2 authorization code fields")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id_token":"signed-token"}`))
	}))
	defer server.Close()
	provider := setting.OAuth2Provider{
		Issuer: "https://issuer.example", AuthorizationURL: "https://issuer.example/authorize",
		TokenURL: server.URL, JWKSURL: "https://issuer.example/keys",
		ClientID: "test-client", ClientSecret: "secret",
		RedirectURI: "https://client.example/users/oauth2/callback",
	}
	flow, err := NewOAuth2Client(provider, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	got, err := flow.Exchange(context.Background(), "code", "verifier")
	if err != nil || got != "signed-token" {
		t.Fatalf("unexpected token exchange: %q %v", got, err)
	}
	u, err := url.Parse(provider.AuthorizationURL)
	if err != nil || u.Scheme != "https" {
		t.Fatal("invalid test authorization endpoint")
	}
}

func TestOAuth2ExchangeProviderFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	flow, err := NewOAuth2Client(setting.OAuth2Provider{
		Issuer: "https://issuer.example", AuthorizationURL: "https://issuer.example/authorize",
		TokenURL: server.URL, JWKSURL: "https://issuer.example/keys",
		ClientID: "test-client", ClientSecret: "secret",
		RedirectURI: "https://client.example/users/oauth2/callback",
	}, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = flow.Exchange(context.Background(), "code", "verifier")
	if !errors.Is(err, ErrOAuth2ProviderUnavailable) {
		t.Fatalf("provider outage not classified: %v", err)
	}
}

func TestOAuth2PKCEStateIsSingleUse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if client.Ping(ctx).Err() != nil {
		t.Skip("Redis required for state integration")
	}
	provider := setting.OAuth2Provider{
		Issuer:           "https://issuer.example",
		AuthorizationURL: "https://issuer.example/authorize",
		TokenURL:         "https://issuer.example/token",
		JWKSURL:          "https://issuer.example/keys",
		ClientID:         "client",
		ClientSecret:     "secret",
		RedirectURI:      "https://client.example/users/oauth2/callback",
	}
	flow, err := NewOAuth2Client(provider, client, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	start, err := flow.Begin(ctx, "login", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	link, err := url.Parse(start.AuthorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	q := link.Query()
	if q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" ||
		q.Get("nonce") == "" || q.Get("state") != start.State {
		t.Fatalf("invalid authorization request: %s", link.RawQuery)
	}
	pending, err := flow.Consume(ctx, start.State)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(pending.Verifier))
	if q.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(digest[:]) ||
		q.Get("nonce") != pending.Nonce || pending.Issuer != provider.Issuer {
		t.Fatal("authorization request not bound to server state")
	}
	if _, err := flow.Consume(ctx, start.State); !errors.Is(err, ErrOAuth2StateExpired) {
		t.Fatalf("consumed authorization state replay accepted: %v", err)
	}
}
