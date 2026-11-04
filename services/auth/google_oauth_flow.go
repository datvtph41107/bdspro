package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/go-redis/redis/v8"
)

const (
	googleAuthorizationURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL         = "https://oauth2.googleapis.com/token"
	googleStateKeyPrefix   = "google:oauth:state:"
	googleOAuthTTL         = 5 * time.Minute
)

var (
	ErrGoogleOAuthProviderUnavailable = errors.New("Google OAuth provider unavailable")
	ErrGoogleOAuthStateExpired        = errors.New("Google OAuth state expired")
	ErrGoogleOAuthStateInvalid        = errors.New("Google OAuth state invalid")
)

// GoogleOAuthPending is the one-use, server-side state of an OAuth interaction.
// Cookies and HTTP response semantics remain in the Gin handler.
type GoogleOAuthPending struct {
	Purpose   string `json:"purpose"`
	AccountID int64  `json:"account_id"`
	Verifier  string `json:"verifier"`
	Nonce     string `json:"nonce"`
	Phone     string `json:"phone,omitempty"`
}

type GoogleOAuthStart struct {
	URL    string
	State  string
	MaxAge int
}

// GoogleOAuthFlow owns state issuance, atomic consumption and the provider
// token request. It has no Gin or public HTTP error dependency.
type GoogleOAuthFlow struct {
	config setting.Config
	redis  *redis.Client
	client *http.Client
}

func NewGoogleOAuthFlow(config setting.Config, redisClient *redis.Client, httpClient *http.Client) *GoogleOAuthFlow {
	return &GoogleOAuthFlow{config: config, redis: redisClient, client: httpClient}
}

func googleRandom() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// Begin preserves the existing Redis state namespace, JSON fields, TTL and
// S256 PKCE challenge. Authorization and proof checks happen before this call.
func (f *GoogleOAuthFlow) Begin(ctx context.Context, purpose string, accountID int64, phone string) (GoogleOAuthStart, error) {
	var empty GoogleOAuthStart
	state, err := googleRandom()
	if err != nil {
		return empty, err
	}
	verifier, err := googleRandom()
	if err != nil {
		return empty, err
	}
	nonce, err := googleRandom()
	if err != nil {
		return empty, err
	}
	record, err := json.Marshal(GoogleOAuthPending{
		Purpose: purpose, AccountID: accountID, Verifier: verifier, Nonce: nonce, Phone: phone,
	})
	if err != nil {
		return empty, err
	}
	if err := f.redis.Set(ctx, googleStateKeyPrefix+state, record, googleOAuthTTL).Err(); err != nil {
		return empty, err
	}
	hash := sha256.Sum256([]byte(verifier))
	params := url.Values{
		"client_id":             {f.config.GoogleClientID},
		"redirect_uri":          {f.config.GoogleRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(hash[:])},
		"code_challenge_method": {"S256"},
	}
	return GoogleOAuthStart{
		URL: googleAuthorizationURL + "?" + params.Encode(),
		State: state, MaxAge: int(googleOAuthTTL.Seconds()),
	}, nil
}

// Consume atomically deletes the state before processing any provider callback.
// Replays are rejected even after token-exchange or identity-verification errors.
func (f *GoogleOAuthFlow) Consume(ctx context.Context, state string) (GoogleOAuthPending, error) {
	var empty GoogleOAuthPending
	data, err := f.redis.GetDel(ctx, googleStateKeyPrefix+state).Bytes()
	if errors.Is(err, redis.Nil) {
		return empty, ErrGoogleOAuthStateExpired
	}
	if err != nil {
		return empty, err
	}
	var pending GoogleOAuthPending
	if json.Unmarshal(data, &pending) != nil || pending.Verifier == "" || pending.Nonce == "" {
		return empty, ErrGoogleOAuthStateInvalid
	}
	return pending, nil
}

// Exchange executes the OAuth authorization-code exchange, preserving its
// prior HTTP status classification and bounded provider response decoding.
func (f *GoogleOAuthFlow) Exchange(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {f.config.GoogleClientID},
		"client_secret": {f.config.GoogleClientSecret},
		"redirect_uri":  {f.config.GoogleRedirectURI},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGoogleOAuthProviderUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= http.StatusInternalServerError {
		return "", ErrGoogleOAuthProviderUnavailable
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New("Google exchange rejected")
	}
	var body struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&body); err != nil {
		return "", fmt.Errorf("%w: %w", ErrGoogleOAuthProviderUnavailable, err)
	}
	if body.IDToken == "" {
		return "", ErrGoogleOAuthProviderUnavailable
	}
	return body.IDToken, nil
}
