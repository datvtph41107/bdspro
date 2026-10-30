package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/internal/config"
)

type googleTestTransport func(*http.Request) (*http.Response, error)

func (f googleTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGoogleTokenExchangeDistinguishesProviderOutages(t *testing.T) {
	tests := []struct {
		name string
		status int
		body string
		networkError error
		wantUnavailable bool
	}{
		{name: "grant rejected", status: http.StatusBadRequest, body: `{"error":"invalid_grant"}`},
		{name: "provider unavailable", status: http.StatusServiceUnavailable, body: `{"error":"down"}`, wantUnavailable: true},
		{name: "provider throttled", status: http.StatusTooManyRequests, body: `{"error":"slow_down"}`, wantUnavailable: true},
		{name: "malformed provider JSON", status: http.StatusOK, body: "{", wantUnavailable: true},
		{name: "missing ID token", status: http.StatusOK, body: `{}`, wantUnavailable: true},
		{name: "network error", networkError: errors.New("private transport secret"), wantUnavailable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: googleTestTransport(func(req *http.Request) (*http.Response, error) {
				if tt.networkError != nil {
					return nil, tt.networkError
				}
				return &http.Response{
					StatusCode: tt.status,
					Body: io.NopCloser(strings.NewReader(tt.body)),
					Header: make(http.Header),
					Request: req,
				}, nil
			})}
			g := &GoogleHandler{httpClient: client, config: config.Config{
				GoogleClientID: "test-client",
				GoogleClientSecret: "test-secret",
				GoogleRedirectURI: "https://example.test/users/google/callback",
			}}
			_, err := g.exchange(context.Background(), "authorization-code", "verifier")
			if err == nil {
				t.Fatal("expected OAuth token exchange failure")
			}
			if got := errors.Is(err, errGoogleProviderUnavailable); got != tt.wantUnavailable {
				t.Fatalf("provider unavailable = %t, want %t", got, tt.wantUnavailable)
			}
		})
	}
}

func TestGoogleTokenExchangeSuccessfulResponse(t *testing.T) {
	g := &GoogleHandler{
		config: config.Config{GoogleClientID: "test-client"},
		httpClient: &http.Client{Transport: googleTestTransport(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{"id_token":"test-id-token"}`)),
				Header: make(http.Header),
				Request: req,
			}, nil
		})},
	}
	idToken, err := g.exchange(context.Background(), "code", "verifier")
	if err != nil || idToken != "test-id-token" {
		t.Fatalf("token=%q, error=%v", idToken, err)
	}
}
