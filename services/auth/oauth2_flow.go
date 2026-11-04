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

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/go-redis/redis/v8"
)

const (
	oauth2StateKeyPrefix = "oidc:flow:state:"
	oauth2StateTTL       = 5 * time.Minute
)

var (
	ErrOAuth2StateExpired        = errors.New("oauth2 authorization state expired")
	ErrOAuth2StateInvalid        = errors.New("oauth2 authorization state invalid")
	ErrOAuth2ProviderUnavailable = errors.New("oauth2 provider unavailable")
	ErrOAuth2AuthorizationFailed = errors.New("oauth2 authorization failed")
)

// OAuth2Pending is server-side state, consumed exactly once before exchanging
// the code. Cookie/HTTP behavior belongs to the caller, not this service.
type OAuth2Pending struct {
	Issuer    string `json:"issuer"`
	Purpose   string `json:"purpose"`
	AccountID int64  `json:"account_id"`
	Verifier  string `json:"verifier"`
	Nonce     string `json:"nonce"`
	Phone     string `json:"phone,omitempty"`
}

type OAuth2Start struct {
	AuthorizationURL string
	State            string
	MaxAge           int
}

type OAuth2Flow struct {
	provider setting.OAuth2Provider
	redis    *redis.Client
	client   *http.Client
}

func NewOAuth2Flow(provider setting.OAuth2Provider, redisClient *redis.Client, client *http.Client) (*OAuth2Flow, error) {
	if err := provider.Validate(); err != nil {
		return nil, err
	}
	if redisClient == nil || client == nil {
		return nil, errors.New("oauth2 dependencies unavailable")
	}
	return &OAuth2Flow{provider: provider, redis: redisClient, client: client}, nil
}

func oauth2Random() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (f *OAuth2Flow) Begin(ctx context.Context, purpose string, accountID int64, phone string) (OAuth2Start, error) {
	var empty OAuth2Start
	switch purpose {
	case "login":
		if accountID != 0 || phone != "" {
			return empty, ErrOAuth2StateInvalid
		}
	case "link", "phone_bind":
		if accountID <= 0 {
			return empty, ErrOAuth2StateInvalid
		}
	default:
		return empty, ErrOAuth2StateInvalid
	}
	state, err := oauth2Random()
	if err != nil {
		return empty, err
	}
	verifier, err := oauth2Random()
	if err != nil {
		return empty, err
	}
	nonce, err := oauth2Random()
	if err != nil {
		return empty, err
	}
	pending := OAuth2Pending{
		Issuer: f.provider.Issuer, Purpose: purpose, AccountID: accountID,
		Verifier: verifier, Nonce: nonce, Phone: phone,
	}
	data, err := json.Marshal(pending)
	if err != nil {
		return empty, err
	}
	if err := f.redis.Set(ctx, oauth2StateKeyPrefix+state, data, oauth2StateTTL).Err(); err != nil {
		return empty, err
	}
	challenge := sha256.Sum256([]byte(verifier))
	u, _ := url.Parse(f.provider.AuthorizationURL) // checked in constructor
	query := u.Query()
	query.Set("client_id", f.provider.ClientID)
	query.Set("redirect_uri", f.provider.RedirectURI)
	query.Set("response_type", "code")
	query.Set("scope", "openid email profile")
	query.Set("state", state)
	query.Set("nonce", nonce)
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	query.Set("code_challenge_method", "S256")
	u.RawQuery = query.Encode()
	return OAuth2Start{AuthorizationURL: u.String(), State: state, MaxAge: int(oauth2StateTTL.Seconds())}, nil
}

func (f *OAuth2Flow) Consume(ctx context.Context, state string) (OAuth2Pending, error) {
	var empty OAuth2Pending
	if len(state) != 43 {
		return empty, ErrOAuth2StateInvalid
	}
	data, err := f.redis.GetDel(ctx, oauth2StateKeyPrefix+state).Bytes()
	if errors.Is(err, redis.Nil) {
		return empty, ErrOAuth2StateExpired
	}
	if err != nil {
		return empty, err
	}
	var pending OAuth2Pending
	if json.Unmarshal(data, &pending) != nil || pending.Issuer != f.provider.Issuer ||
		pending.Verifier == "" || pending.Nonce == "" {
		return empty, ErrOAuth2StateInvalid
	}
	return pending, nil
}

func (f *OAuth2Flow) Exchange(ctx context.Context, code, verifier string) (string, error) {
	if code == "" || verifier == "" {
		return "", ErrOAuth2AuthorizationFailed
	}
	form := url.Values{
		"code": {code}, "client_id": {f.provider.ClientID},
		"client_secret": {f.provider.ClientSecret},
		"redirect_uri": {f.provider.RedirectURI},
		"grant_type": {"authorization_code"}, "code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.provider.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrOAuth2ProviderUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500 {
		return "", ErrOAuth2ProviderUnavailable
	}
	if res.StatusCode != http.StatusOK {
		return "", ErrOAuth2AuthorizationFailed
	}
	var result struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&result); err != nil {
		return "", fmt.Errorf("%w: %w", ErrOAuth2ProviderUnavailable, err)
	}
	if result.IDToken == "" {
		return "", ErrOAuth2ProviderUnavailable
	}
	return result.IDToken, nil
}

// Verify exchanges no secrets; it checks the trusted issuer's signed ID token.
func (f *OAuth2Flow) Verify(ctx context.Context, idToken, nonce string) (auth_model.OIDCIdentity, error) {
	return (OIDCTokenVerifier{
		Issuer: f.provider.Issuer, Audience: f.provider.ClientID,
		JWKSURL: f.provider.JWKSURL, Client: f.client,
	}).Verify(ctx, idToken, nonce)
}
