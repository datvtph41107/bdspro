package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/go-redis/redis/v8"
)

func TestOAuth2FlowTokenExchange(t *testing.T) {
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
	flow, err := NewOAuth2Flow(provider, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}), server.Client())
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
	flow, err := NewOAuth2Flow(setting.OAuth2Provider{
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
