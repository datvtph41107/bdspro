package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/golang-jwt/jwt/v5"
)

type googleTestTransport func(*http.Request) (*http.Response, error)

func (f googleTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGoogleExchangeContract(t *testing.T) {
	cfg := setting.GoogleSettings{ClientID: "fixture-client", ClientSecret: "fixture-secret", CallbackURL: "https://example.invalid/api/v1/auth/google/callback"}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	claims := googleClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: GoogleIssuer, Subject: "fixture-subject", Audience: jwt.ClaimStrings{cfg.ClientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)), IssuedAt: jwt.NewNumericDate(now),
		},
		Nonce: "fixture-nonce", Email: "fixture@gmail.com", EmailVerified: true, AuthTime: now.Unix(),
	}
	signed := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	signed.Header["kid"] = "fixture-key"
	raw, err := signed.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	jwk := map[string]string{
		"kid": "fixture-key", "kty": "RSA", "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
	jsonBody := func(value any) string {
		b, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	tokenBody := jsonBody(map[string]string{"id_token": raw})
	keysBody := jsonBody(map[string]any{"keys": []any{jwk}})
	for _, tc := range []struct {
		name        string
		tokenStatus int
		tokenBody   string
		keysBody    string
		nonce       string
		calls       int
		want        error
	}{
		{"valid", 200, tokenBody, keysBody, "fixture-nonce", 2, nil},
		{"wrong nonce", 200, tokenBody, keysBody, "wrong", 2, ErrGoogleProof},
		{"unknown kid", 200, tokenBody, `{"keys":[]}`, "fixture-nonce", 2, ErrGoogleProof},
		{"duplicate kid", 200, tokenBody, jsonBody(map[string]any{"keys": []any{jwk, jwk}}), "fixture-nonce", 2, ErrGoogleProof},
		{"invalid code", 400, "private upstream detail", "", "fixture-nonce", 1, ErrGoogleProof},
		{"provider unavailable", 503, "private upstream detail", "", "fixture-nonce", 1, ErrGoogleUnavailable},
		{"redirect refused", 302, "", "", "fixture-nonce", 1, ErrGoogleUnavailable},
		{"oversized JSON", 200, strings.Repeat("x", (128<<10)+1), "", "fixture-nonce", 1, ErrGoogleUnavailable},
		{"malformed JSON", 200, "{", "", "fixture-nonce", 1, ErrGoogleUnavailable},
		{"missing ID token", 200, "{}", "", "fixture-nonce", 1, ErrGoogleProof},
		{"malformed JWKS", 200, tokenBody, "{", "fixture-nonce", 2, ErrGoogleUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{
				CheckRedirect: googleClient.CheckRedirect,
				Transport: googleTestTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					status, body := tc.tokenStatus, tc.tokenBody
					switch calls {
					case 1:
						if r.Method != http.MethodPost || r.URL.String() != googleTokenURL || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
							t.Fatal("incorrect token request")
						}
						if err := r.ParseForm(); err != nil {
							t.Fatal(err)
						}
						for k, v := range map[string]string{
							"grant_type": "authorization_code", "client_id": cfg.ClientID, "client_secret": cfg.ClientSecret,
							"redirect_uri": cfg.CallbackURL, "code": "fixture-code", "code_verifier": "fixture-verifier",
						} {
							if r.PostForm.Get(k) != v {
								t.Fatal("incorrect form field", k)
							}
						}
					case 2:
						if r.Method != http.MethodGet || r.URL.String() != googleKeysURL || r.Header.Get("Authorization") != "" {
							t.Fatal("incorrect JWKS request")
						}
						status, body = 200, tc.keysBody
					default:
						t.Fatal("unexpected provider request")
					}
					header := make(http.Header)
					if status == 302 {
						header.Set("Location", "https://unexpected.example.invalid")
					}
					return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				}),
			}
			got, err := exchangeGoogle(context.Background(), client, cfg, "fixture-code", "fixture-verifier", tc.nonce)
			if !errors.Is(err, tc.want) || calls != tc.calls {
				t.Fatalf("unexpected exchange result: error=%v calls=%d", err, calls)
			}
			if tc.want == nil && (got.Subject != claims.Subject || got.Email != claims.Email || !got.AuthoritativeEmail || got.AuthTime != claims.AuthTime) {
				t.Fatal("verified identity contract changed")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{Transport: googleTestTransport(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})}
	if _, err := exchangeGoogle(ctx, client, cfg, "code", "verifier", "nonce"); !errors.Is(err, ErrGoogleUnavailable) {
		t.Fatal("canceled exchange did not fail closed", err)
	}
}

func TestGoogleExchangeCapacity(t *testing.T) {
	if cap(googleSlots) != 4 || len(googleSlots) != 0 {
		t.Fatal("unexpected Google capacity")
	}
	for i := 0; i < cap(googleSlots); i++ {
		googleSlots <- struct{}{}
	}
	defer func() {
		for len(googleSlots) > 0 {
			<-googleSlots
		}
	}()
	cfg := setting.GoogleSettings{ClientID: "fixture-client"}
	if _, err := ExchangeGoogle(context.Background(), cfg, "code", "verifier", "nonce"); !errors.Is(err, ErrGoogleUnavailable) {
		t.Fatal("saturated exchange bypassed admission", err)
	}
}
