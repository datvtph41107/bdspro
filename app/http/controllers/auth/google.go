package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	apperrors "github.com/datvtph41107/bdspro/app/errors"
	middleware "github.com/datvtph41107/bdspro/app/http/middleware"
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"github.com/datvtph41107/bdspro/app/http/resources"
	"github.com/datvtph41107/bdspro/app/http/responses"
	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
)

const (
	googleAuthorizationURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL         = "https://oauth2.googleapis.com/token"
	googleOAuthCookie      = "bdspro_google_state"
	googleOAuthTTL         = 5 * time.Minute
)

type GoogleController struct {
	store        db.Store
	maker        token.Maker
	config       util.Config
	redis        *redis.Client
	httpClient   *http.Client
	secureCookie bool
}
type googlePending struct {
	Purpose   string `json:"purpose"`
	AccountID int64  `json:"account_id"`
	Verifier  string `json:"verifier"`
	Nonce     string `json:"nonce"`
}

func NewGoogleController(
	store db.Store, maker token.Maker, config util.Config,
	client *redis.Client, httpClient *http.Client, secureCookie bool,
) *GoogleController {
	return &GoogleController{
		store: store, maker: maker, config: config,
		redis: client, httpClient: httpClient, secureCookie: secureCookie,
	}
}

type googleLinkStartRequest = authrequest.GoogleLinkBody

func (g *GoogleController) StartLink(ctx *gin.Context) {
	var req googleLinkStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		responses.InvalidInput(ctx, err, "invalid request")
		return
	}
	accountID := middleware.AccountID(ctx)
	account, err := g.store.GetAccountByID(ctx.Request.Context(), accountID)
	if err != nil || util.CheckPassword(req.Password, account.PasswordHash) != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	authURL, err := g.begin(ctx, "link", accountID)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google sign-in unavailable"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"authorization_url": authURL})
}

func (g *GoogleController) StartLogin(ctx *gin.Context) {
	authURL, err := g.begin(ctx, "login", 0)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google sign-in unavailable"})
		return
	}
	ctx.Redirect(http.StatusFound, authURL)
}

func googleRandom() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (g *GoogleController) stateCookie(ctx *gin.Context, state string, age int) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name: googleOAuthCookie, Value: state, Path: "/users/google/callback", MaxAge: age,
		HttpOnly: true, Secure: g.secureCookie, SameSite: http.SameSiteLaxMode,
	})
}

func (g *GoogleController) begin(ctx *gin.Context, purpose string, accountID int64) (string, error) {
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
	record, err := json.Marshal(googlePending{Purpose: purpose, AccountID: accountID, Verifier: verifier, Nonce: nonce})
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

func (g *GoogleController) Callback(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	state := ctx.Query("state")
	cookie, err := ctx.Request.Cookie(googleOAuthCookie)
	if err != nil || state == "" || len(state) != 43 || subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid Google authorization state"})
		return
	}
	g.stateCookie(ctx, "", -1)
	data, err := g.redis.GetDel(ctx.Request.Context(), "google:oauth:state:"+state).Bytes()
	if errors.Is(err, redis.Nil) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "expired Google authorization state"})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google sign-in unavailable"})
		return
	}
	var pending googlePending
	if json.Unmarshal(data, &pending) != nil || pending.Verifier == "" || pending.Nonce == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid Google authorization state"})
		return
	}
	if ctx.Query("error") != "" || ctx.Query("code") == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Google authorization declined"})
		return
	}
	idToken, err := g.exchange(ctx.Request.Context(), ctx.Query("code"), pending.Verifier)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Google authorization failed"})
		return
	}
	subject, err := auth.VerifyGoogleIDToken(ctx.Request.Context(), idToken, g.config.GoogleClientID, pending.Nonce)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Google identity"})
		return
	}

	if pending.Purpose == "link" && pending.AccountID > 0 {
		err = g.store.LinkGoogleIdentity(ctx.Request.Context(), db.LinkGoogleIdentityParams{AccountID: pending.AccountID, GoogleSub: subject})
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusConflict, gin.H{"error": "Google identity already linked"})
			return
		}
		if pgErr := apperrors.Postgres(err); pgErr != nil && pgErr.Code == "23505" {
			ctx.JSON(http.StatusConflict, gin.H{"error": "account already linked"})
			return
		}
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "linking failed"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"linked": true})
		return
	}
	if pending.Purpose != "login" || pending.AccountID != 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid Google authorization state"})
		return
	}
	accountID, err := g.store.GetGoogleAccountID(ctx.Request.Context(), subject)
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Google identity not linked"})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	account, err := g.store.GetAccountByID(ctx.Request.Context(), accountID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "account unavailable"})
		return
	}
	result, err := auth.IssueSession(ctx.Request.Context(), g.store, g.maker, g.config, account, ctx.Request.UserAgent(), ctx.ClientIP())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	ctx.JSON(http.StatusOK, resources.NewLogin(result))
}

func (g *GoogleController) exchange(ctx context.Context, code, verifier string) (string, error) {
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
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", errors.New("Google exchange failed")
	}
	var body struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&body); err != nil {
		return "", err
	}
	if body.IDToken == "" {
		return "", errors.New("missing Google ID token")
	}
	return body.IDToken, nil
}
