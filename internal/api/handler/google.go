package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	middleware "github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/datvtph41107/bdspro/internal/config"
	"github.com/datvtph41107/bdspro/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

const (
	googleAuthorizationURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL         = "https://oauth2.googleapis.com/token"
	googleOAuthCookie      = "bdspro_google_state"
	googleOAuthTTL         = 5 * time.Minute
)

var errGoogleProviderUnavailable = errors.New("Google OAuth provider unavailable")

type GoogleHandler struct {
	auth         *service.GoogleAuthService
	config       config.Config
	redis        *redis.Client
	httpClient   *http.Client
	secureCookie bool
	phone        *service.PhoneOTP
}
type googlePending struct {
	Purpose   string `json:"purpose"`
	AccountID int64  `json:"account_id"`
	Verifier  string `json:"verifier"`
	Nonce     string `json:"nonce"`
	Phone     string `json:"phone,omitempty"`
}

func NewGoogleHandler(
	auth *service.GoogleAuthService, config config.Config,
	client *redis.Client, httpClient *http.Client, secureCookie bool,
) *GoogleHandler {
	return &GoogleHandler{
		auth: auth, config: config,
		redis: client, httpClient: httpClient, secureCookie: secureCookie,
	}
}

type googleLinkStartRequest = GoogleLinkRequest

func (g *GoogleHandler) StartLink(ctx *gin.Context) {
	var req googleLinkStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Invalid(ctx, err)
		return
	}
	accountID := middleware.AccountID(ctx)
	if (req.Password == "") == (req.PhoneCode == "") {
		response.Fail(ctx, response.InvalidLinkProof)
		return
	}
	if req.PhoneCode != "" {
		if g.phone == nil {
			response.Fail(ctx, response.PhoneReauthenticationUnavailable)
			return
		}
		if err := g.phone.AuthorizePhoneProof(ctx.Request.Context(), accountID, req.PhoneCode); err != nil {
			response.Error(ctx, err)
			return
		}
	} else if err := g.auth.CheckPassword(ctx.Request.Context(), accountID, req.Password); err != nil {
		response.Error(ctx, err)
		return
	}
	authURL, err := g.begin(ctx, "link", accountID, "")
	if err != nil {
		googleUnavailable(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"authorization_url": authURL})
}

func (g *GoogleHandler) StartLogin(ctx *gin.Context) {
	authURL, err := g.begin(ctx, "login", 0, "")
	if err != nil {
		googleUnavailable(ctx, err)
		return
	}
	ctx.Redirect(http.StatusFound, authURL)
}

// SetPhoneBinding enables the Google reauthentication path in environments
// where phone verification has an actual delivery implementation.
func (g *GoogleHandler) SetPhoneBinding(phone service.PhoneOTP) {
	g.phone = &phone
}

// StartPhoneBind starts an OAuth reauthentication, not a Google account link.
// Only the linked Google subject can authorize a phone-binding challenge.
func (g *GoogleHandler) StartPhoneBind(ctx *gin.Context) {
	if g.phone == nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Invalid(ctx, err)
		return
	}
	authURL, err := g.begin(ctx, "phone_bind", middleware.AccountID(ctx), req.Phone)
	if err != nil {
		googleUnavailable(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"authorization_url": authURL})
}

func googleRandom() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (g *GoogleHandler) stateCookie(ctx *gin.Context, state string, age int) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name: googleOAuthCookie, Value: state, Path: "/users/google/callback", MaxAge: age,
		HttpOnly: true, Secure: g.secureCookie, SameSite: http.SameSiteLaxMode,
	})
}

func (g *GoogleHandler) begin(ctx *gin.Context, purpose string, accountID int64, phone string) (string, error) {
	state, err := googleRandom()
	if err != nil {
		return "", err
	}
	verifier, err := googleRandom()
	if err != nil {
		return "", err
	}
	nonce, err := googleRandom()
	if err != nil {
		return "", err
	}
	record, err := json.Marshal(googlePending{Purpose: purpose, AccountID: accountID, Verifier: verifier, Nonce: nonce, Phone: phone})
	if err != nil {
		return "", err
	}
	if err = g.redis.Set(ctx.Request.Context(), "google:oauth:state:"+state, record, googleOAuthTTL).Err(); err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(verifier))
	params := url.Values{
		"client_id":             {g.config.GoogleClientID},
		"redirect_uri":          {g.config.GoogleRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(hash[:])},
		"code_challenge_method": {"S256"},
	}
	g.stateCookie(ctx, state, int(googleOAuthTTL.Seconds()))
	ctx.Header("Cache-Control", "no-store")
	return googleAuthorizationURL + "?" + params.Encode(), nil
}

func (g *GoogleHandler) Callback(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	state := ctx.Query("state")
	cookie, err := ctx.Request.Cookie(googleOAuthCookie)
	if err != nil || state == "" || len(state) != 43 || subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		response.Fail(ctx, response.GoogleInvalidState)
		return
	}
	g.stateCookie(ctx, "", -1)
	data, err := g.redis.GetDel(ctx.Request.Context(), "google:oauth:state:"+state).Bytes()
	if errors.Is(err, redis.Nil) {
		response.Fail(ctx, response.GoogleStateExpired)
		return
	}
	if err != nil {
		googleUnavailable(ctx, err)
		return
	}
	var pending googlePending
	if json.Unmarshal(data, &pending) != nil || pending.Verifier == "" || pending.Nonce == "" {
		response.Fail(ctx, response.GoogleInvalidState)
		return
	}
	if ctx.Query("error") != "" || ctx.Query("code") == "" {
		response.Fail(ctx, response.GoogleAuthorizationDeclined)
		return
	}
	idToken, err := g.exchange(ctx.Request.Context(), ctx.Query("code"), pending.Verifier)
	if errors.Is(err, errGoogleProviderUnavailable) {
		googleUnavailable(ctx, err)
		return
	}
	if err != nil {
		response.Fail(ctx, response.GoogleAuthorizationFailed)
		return
	}
	identity, err := service.VerifyGoogleIDToken(ctx.Request.Context(), idToken, g.config.GoogleClientID, pending.Nonce)
	if err != nil {
		response.Fail(ctx, response.GoogleInvalidIdentity)
		return
	}

	if pending.Purpose == "phone_bind" && pending.AccountID > 0 && g.phone != nil {
		if err := g.phone.StartGoogleBind(
			ctx.Request.Context(), pending.AccountID, identity.Subject, pending.Phone,
		); err != nil {
			response.Error(ctx, err)
			return
		}
		ctx.JSON(http.StatusAccepted, gin.H{"phone_verification_started": true})
		return
	}
	if pending.Purpose == "link" && pending.AccountID > 0 {
		if err := g.auth.Link(ctx.Request.Context(), pending.AccountID, identity.Subject); err != nil {
			response.Error(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"linked": true})
		return
	}
	if pending.Purpose != "login" || pending.AccountID != 0 {
		response.Fail(ctx, response.GoogleInvalidState)
		return
	}
	result, err := g.auth.Login(ctx.Request.Context(), identity, ctx.Request.UserAgent(), ctx.ClientIP())
	if err != nil {
		response.Error(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newLoginResponse(result))
}

func (g *GoogleHandler) exchange(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {g.config.GoogleClientID},
		"client_secret": {g.config.GoogleClientSecret},
		"redirect_uri":  {g.config.GoogleRedirectURI},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errGoogleProviderUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= http.StatusInternalServerError {
		return "", errGoogleProviderUnavailable
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New("Google exchange rejected")
	}
	var body struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&body); err != nil {
		return "", fmt.Errorf("%w: %w", errGoogleProviderUnavailable, err)
	}
	if body.IDToken == "" {
		return "", errGoogleProviderUnavailable
	}
	return body.IDToken, nil
}

// googleUnavailable records a bounded diagnostic without logging OAuth codes,
// access tokens, provider response bodies or raw network error messages.
func googleUnavailable(c *gin.Context, err error) {
	log.Ctx(c.Request.Context()).Error().
		Str("dependency", "google_oauth").
		Str("error_type", fmt.Sprintf("%T", err)).
		Msg("auth.provider_unavailable")
	response.Fail(c, response.GoogleUnavailable)
}
