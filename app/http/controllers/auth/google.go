package auth

import (
    "crypto/subtle"
    "encoding/json"
    "errors"
    "net/http"

    apperrors "github.com/datvtph41107/bdspro/app/errors"
    "github.com/datvtph41107/bdspro/app/http/request"
    middleware "github.com/datvtph41107/bdspro/app/http/middleware"
    authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
    "github.com/datvtph41107/bdspro/app/http/resources"
    "github.com/datvtph41107/bdspro/auth"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/token"
    "github.com/datvtph41107/bdspro/util"
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "github.com/jackc/pgx/v5"
)

type GoogleController struct {
	store        db.Store
	maker        token.Maker
	config       util.Config
	redis        *redis.Client
	httpClient   *http.Client
	secureCookie bool
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
	if !request.JSON(ctx, &req) {
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
