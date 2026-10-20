package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTwilioSMSHTTPWireAndFailureRedaction(t *testing.T) {
	calls := 0
	provider := TwilioSMS{
		AccountSID: "AC" + strings.Repeat("a", 32),
		AuthToken:  "token-secret",
		From:       "+15017122661",
		Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Scheme != "https" || r.URL.Host != "api.twilio.com" || r.Method != "POST" {
				t.Fatal("unexpected SMS destination")
			}
			user, pass, ok := r.BasicAuth()
			if !ok || user != "AC"+strings.Repeat("a", 32) || pass != "token-secret" {
				t.Fatal("wrong provider auth")
			}
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), "To=%2B84901234567") || !strings.Contains(string(raw), "Body=") {
				t.Fatal("missing encoded SMS destination or body")
			}
			return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
		})},
	}
	if err := provider.SendOTP(context.Background(), "+84901234567", "123456"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	provider.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("token-secret")
	})}
	if err := provider.SendOTP(context.Background(), "+84901234567", "123456"); err == nil || strings.Contains(err.Error(), "token-secret") {
		t.Fatalf("provider leaked credentials: %v", err)
	}
}
