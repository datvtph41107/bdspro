package auth

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/golang-jwt/jwt/v5"
)

const (
	GoogleIssuer = "https://accounts.google.com"

	// Shared by ID-token freshness validation and database proof issuance.
	GoogleAuthFreshnessSeconds = 300
	GoogleAuthFutureSkewSeconds = 30
)
const googleTokenURL = "https://oauth2.googleapis.com/token"
const googleKeysURL = "https://www.googleapis.com/oauth2/v3/certs"

var ErrGoogleProof = errors.New("invalid Google identity proof")
var ErrGoogleUnavailable = errors.New("Google provider unavailable")

type GoogleIdentity struct {
	Subject            string
	Email              string
	AuthoritativeEmail bool
	AuthTime           int64
}

type googleClaims struct {
	jwt.RegisteredClaims
	Nonce         string `json:"nonce"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HostedDomain  string `json:"hd"`
	Authorized    string `json:"azp"`
	AuthTime      int64  `json:"auth_time"`
}

func GoogleAuthorizationURL(cfg setting.GoogleSettings, state, nonce, verifier string) string {
	challenge := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {cfg.CallbackURL},
		"response_type":         {"code"},
		"scope":                 {"openid email"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	return GoogleIssuer + "/o/oauth2/v2/auth?" + q.Encode()
}

func GoogleStepUpURL(authorizationURL string) (string, error) {
	u, err := url.Parse(authorizationURL)
	if err != nil || u.Scheme != "https" || u.Host != "accounts.google.com" || u.Path != "/o/oauth2/v2/auth" {
		return "", ErrGoogleProof
	}
	q := u.Query()
	q.Set("claims", `{"id_token":{"auth_time":{"essential":true}}}`)
	q.Set("prompt", "select_account")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func GoogleAuthTimeRecent(authTime int64, now time.Time) bool {
	if authTime <= 0 {
		return false
	}
	at := time.Unix(authTime, 0)
	return at.After(now.Add(-GoogleAuthFreshnessSeconds*time.Second)) && !at.After(now.Add(GoogleAuthFutureSkewSeconds*time.Second))
}

var googleClient = &http.Client{
	Timeout: 5 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

var googleSlots = make(chan struct{}, 4)

func ExchangeGoogle(ctx context.Context, cfg setting.GoogleSettings, code, verifier, nonce string) (GoogleIdentity, error) {
	if !cfg.Enabled() || len(code) == 0 || len(code) > 4096 {
		return GoogleIdentity{}, ErrGoogleProof
	}
	select {
	case googleSlots <- struct{}{}:
		defer func() { <-googleSlots }()
	default:
		return GoogleIdentity{}, ErrGoogleUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.CallbackURL},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return GoogleIdentity{}, ErrGoogleUnavailable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var response struct {
		IDToken string `json:"id_token"`
	}
	if err := googleJSON(googleClient, req, &response); err != nil {
		return GoogleIdentity{}, err
	}
	if len(response.IDToken) == 0 || len(response.IDToken) > 16384 {
		return GoogleIdentity{}, ErrGoogleProof
	}
	keyReq, err := http.NewRequestWithContext(ctx, http.MethodGet, googleKeysURL, nil)
	if err != nil {
		return GoogleIdentity{}, ErrGoogleUnavailable
	}
	var set googleKeySet
	if err := googleJSON(googleClient, keyReq, &set); err != nil {
		return GoogleIdentity{}, err
	}
	return verifyGoogleIDToken(response.IDToken, cfg.ClientID, nonce, set.key)
}

func googleJSON(client *http.Client, req *http.Request, out any) error {
	resp, err := client.Do(req)
	if err != nil {
		return ErrGoogleUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			return ErrGoogleProof
		}
		return ErrGoogleUnavailable
	}
	const limit = 128 << 10
	raw, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(raw) > limit || json.Unmarshal(raw, out) != nil {
		return ErrGoogleUnavailable
	}
	return nil
}

type googleKeySet struct {
	Keys []struct {
		ID   string `json:"kid"`
		Type string `json:"kty"`
		Use  string `json:"use"`
		Alg  string `json:"alg"`
		N    string `json:"n"`
		E    string `json:"e"`
	} `json:"keys"`
}

func (set googleKeySet) key(t *jwt.Token) (any, error) {
	kid, ok := t.Header["kid"].(string)
	if !ok || kid == "" || len(kid) > 256 {
		return nil, ErrGoogleProof
	}
	var result *rsa.PublicKey
	for _, k := range set.Keys {
		if k.ID != kid {
			continue
		}
		if result != nil || k.Type != "RSA" || k.Use != "sig" || k.Alg != "RS256" {
			return nil, ErrGoogleProof
		}
		n, err := base64.RawURLEncoding.Strict().DecodeString(k.N)
		if err != nil || len(n) < 256 || len(n) > 1024 {
			return nil, ErrGoogleProof
		}
		e, err := base64.RawURLEncoding.Strict().DecodeString(k.E)
		if err != nil || len(e) == 0 || len(e) > 4 {
			return nil, ErrGoogleProof
		}
		exp := new(big.Int).SetBytes(e).Int64()
		modulus := new(big.Int).SetBytes(n)
		if exp < 3 || exp > 2147483647 || exp%2 == 0 || modulus.BitLen() < 2048 || modulus.Bit(0) == 0 {
			return nil, ErrGoogleProof
		}
		result = &rsa.PublicKey{N: modulus, E: int(exp)}
	}
	if result == nil {
		return nil, ErrGoogleProof
	}
	return result, nil
}

func verifyGoogleIDToken(raw, clientID, nonce string, key jwt.Keyfunc) (GoogleIdentity, error) {
	claims := &googleClaims{}
	t, err := jwt.ParseWithClaims(raw, claims, key,
		jwt.WithValidMethods([]string{"RS256"}), jwt.WithAudience(clientID),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !t.Valid || (claims.Issuer != GoogleIssuer && claims.Issuer != "accounts.google.com") ||
		claims.IssuedAt == nil || claims.Subject == "" || len(claims.Subject) > 255 ||
		nonce == "" || claims.Nonce != nonce || len(claims.Email) > 254 ||
		(len(claims.Audience) > 1 && claims.Authorized != clientID) ||
		(claims.Authorized != "" && claims.Authorized != clientID) {
		return GoogleIdentity{}, ErrGoogleProof
	}
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	authoritative := claims.EmailVerified && (strings.HasSuffix(email, "@gmail.com") || claims.HostedDomain != "")
	return GoogleIdentity{Subject: claims.Subject, Email: email, AuthoritativeEmail: authoritative, AuthTime: claims.AuthTime}, nil
}
