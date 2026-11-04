package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/go-redis/redis/v8"
)

type googleFlowTestTransport func(*http.Request) (*http.Response, error)

func (f googleFlowTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGoogleOAuthExchangeClassifiesProviderFailures(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		body            string
		networkError    error
		wantUnavailable bool
	}{
		{name: "grant rejected", status: http.StatusBadRequest, body: `{"error":"invalid_grant"}`},
		{name: "provider unavailable", status: http.StatusServiceUnavailable, body: `{"error":"down"}`, wantUnavailable: true},
		{name: "provider throttled", status: http.StatusTooManyRequests, body: `{"error":"slow_down"}`, wantUnavailable: true},
		{name: "malformed JSON", status: http.StatusOK, body: "{", wantUnavailable: true},
		{name: "missing ID token", status: http.StatusOK, body: "{}", wantUnavailable: true},
		{name: "network error", networkError: errors.New("private transport secret"), wantUnavailable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: googleFlowTestTransport(func(req *http.Request) (*http.Response, error) {
				if tt.networkError != nil {
					return nil, tt.networkError
				}
				return &http.Response{
					StatusCode: tt.status,
					Body:       io.NopCloser(strings.NewReader(tt.body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			})}
			flow := NewGoogleOAuthFlow(setting.Config{
				GoogleClientID: "test-client", GoogleClientSecret: "test-secret",
				GoogleRedirectURI: "https://example.test/users/google/callback",
			}, nil, client)
			_, err := flow.Exchange(context.Background(), "authorization-code", "verifier")
			if err == nil {
				t.Fatal("expected provider exchange failure")
			}
			if got := errors.Is(err, ErrGoogleOAuthProviderUnavailable); got != tt.wantUnavailable {
				t.Fatalf("unavailable=%t, want %t", got, tt.wantUnavailable)
			}
		})
	}
}

func TestGoogleOAuthExchangeSuccessfulResponse(t *testing.T) {
	client := &http.Client{Transport: googleFlowTestTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != googleTokenURL {
			t.Fatalf("wrong token exchange endpoint: %s %s", req.Method, req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id_token":"test-id-token"}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	flow := NewGoogleOAuthFlow(setting.Config{GoogleClientID: "test-client"}, nil, client)
	token, err := flow.Exchange(context.Background(), "code", "verifier")
	if err != nil || token != "test-id-token" {
		t.Fatalf("token=%q, error=%v", token, err)
	}
}

func TestGoogleOAuthStateSingleUseAndPKCE(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis integration unavailable: %v", err)
	}
	cfg := setting.Config{
		GoogleClientID: "test-client", GoogleRedirectURI: "https://example.test/users/google/callback",
	}
	flow := NewGoogleOAuthFlow(cfg, client, &http.Client{Timeout: time.Second})
	start, err := flow.Begin(ctx, "phone_bind", 77, "+84901234567")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Del(context.Background(), googleStateKeyPrefix+start.State)
	if len(start.State) != 43 || start.MaxAge != 300 {
		t.Fatalf("invalid state or TTL: %+v", start)
	}
	authURL, err := url.Parse(start.URL)
	if err != nil {
		t.Fatal(err)
	}
	q := authURL.Query()
	if q.Get("state") != start.State || q.Get("code_challenge_method") != "S256" ||
		q.Get("code_challenge") == "" || q.Get("nonce") == "" ||
		q.Get("redirect_uri") != cfg.GoogleRedirectURI {
		t.Fatalf("invalid OAuth params: %v", q)
	}
	if ttl := client.TTL(ctx, googleStateKeyPrefix+start.State).Val(); ttl <= 0 || ttl > googleOAuthTTL {
		t.Fatalf("wrong persisted state TTL: %v", ttl)
	}
	pending, err := flow.Consume(ctx, start.State)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Purpose != "phone_bind" || pending.AccountID != 77 || pending.Phone != "+84901234567" ||
		pending.Nonce != q.Get("nonce") || pending.Verifier == "" {
		t.Fatalf("consumed incorrect pending state: %+v", pending)
	}
	if _, err := flow.Consume(ctx, start.State); !errors.Is(err, ErrGoogleOAuthStateExpired) {
		t.Fatalf("replayed state accepted: %v", err)
	}
}

func TestGoogleOAuthMalformedStateIsRejected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis integration unavailable: %v", err)
	}
	flow := NewGoogleOAuthFlow(setting.Config{}, client, nil)
	start, err := flow.Begin(ctx, "login", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, googleStateKeyPrefix+start.State, "{", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Consume(ctx, start.State); !errors.Is(err, ErrGoogleOAuthStateInvalid) {
		t.Fatalf("malformed state accepted: %v", err)
	}
}
