package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	apperrors "github.com/datvtph41107/bdspro/core/errors"
	"github.com/datvtph41107/bdspro/core/middleware"
	authrequest "github.com/datvtph41107/bdspro/api/handler/request/auth"
	"github.com/datvtph41107/bdspro/api/handler/resource"
	"github.com/datvtph41107/bdspro/core/response"
	"github.com/datvtph41107/bdspro/api/service"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/config"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// AuthHandler is the HTTP entrypoint. It owns no database connection,
// SQL string, transaction runtime or queue implementation.
type AuthHandler struct {
	store     db.Store
	scheduler service.VerificationTaskScheduler
	maker     token.Maker
	config    config.Config
	limiter   *middleware.IdentityLimiter
}

func NewAuthHandler(store db.Store, scheduler service.VerificationTaskScheduler, maker token.Maker, config config.Config, limiter *middleware.IdentityLimiter) *AuthHandler {
	return &AuthHandler{store: store, scheduler: scheduler, maker: maker, config: config, limiter: limiter}
}

func (c *AuthHandler) Register(ctx *gin.Context) {
	var input authrequest.RegisterBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if !c.limiter.Allow(ctx, "register", input.Email, 10, 3, time.Hour) {
		return
	}
	account, err := service.Register(ctx.Request.Context(), c.store, c.scheduler, input.Email, input.Password)
	response.Send(ctx, http.StatusCreated, resources.NewUser(account), err, "registration failed")
}

func (c *AuthHandler) Login(ctx *gin.Context) {
	var input authrequest.LoginBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if !c.limiter.Allow(ctx, "login", input.Email, 30, 10, time.Minute) {
		return
	}
	result, err := service.Login(ctx.Request.Context(), c.store, c.maker, c.config,
		input.Email, input.Password, ctx.Request.UserAgent(), ctx.ClientIP())
	response.Send(ctx, http.StatusOK, resources.NewLogin(result), err, "login failed")
}

func (c *AuthHandler) VerifyEmail(ctx *gin.Context) {
	var input authrequest.VerifyEmailQuery
	if err := ctx.ShouldBindQuery(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid verification link")
		return
	}
	if !c.limiter.Allow(ctx, "verify_email", strconv.FormatInt(input.EmailID, 10), 60, 10, time.Minute) {
		return
	}
	_, err := c.store.VerifyEmailTx(ctx.Request.Context(), db.ConsumeVerifyEmailParams{
		ID: input.EmailID, SecretCode: input.SecretCode,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		response.Send(ctx, 0, nil, apperrors.InvalidVerificationLink, "")
		return
	}
	response.Send(ctx, http.StatusOK, gin.H{"verified": true}, err, "verification failed")
}

func (c *AuthHandler) Renew(ctx *gin.Context) {
	var input authrequest.TokenBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if !c.limiter.Allow(ctx, "refresh", input.RefreshToken, 60, 20, time.Minute) {
		return
	}
	payload, err := c.maker.VerifyToken(input.RefreshToken, token.TokenTypeRefresh)
	if err != nil {
		response.Send(ctx, 0, nil, apperrors.InvalidRefreshToken, "")
		return
	}
	session, err := c.store.GetSession(ctx.Request.Context(), payload.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.Send(ctx, 0, nil, apperrors.SessionUnavailable, "")
		return
	}
	if err != nil {
		response.Send(ctx, 0, nil, err, "renew failed")
		return
	}
	if session.IsBlocked || session.AccountID != payload.AccountID ||
		session.RefreshToken != input.RefreshToken || !time.Now().Before(session.ExpiresAt) {
		response.Send(ctx, 0, nil, apperrors.SessionUnavailable, "")
		return
	}
	access, tokenPayload, err := c.maker.CreateToken(
		payload.AccountID, c.config.AccessTokenDuration, token.TokenTypeAccess,
	)
	if err != nil {
		response.Send(ctx, 0, nil, err, "renew failed")
		return
	}
	response.Send(ctx, http.StatusOK, resources.RenewAccessToken{
		AccessToken: access, AccessTokenExpiresAt: tokenPayload.ExpiredAt,
	}, nil, "renew failed")
}

func (c *AuthHandler) Revoke(ctx *gin.Context) {
	var input authrequest.TokenBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if !c.limiter.Allow(ctx, "revoke", input.RefreshToken, 60, 20, time.Minute) {
		return
	}
	payload, err := c.maker.VerifyToken(input.RefreshToken, token.TokenTypeRefresh)
	if err != nil {
		response.Send(ctx, 0, nil, apperrors.InvalidRefreshToken, "")
		return
	}
	_, err = c.store.BlockSession(ctx.Request.Context(), db.BlockSessionParams{
		ID: payload.ID, AccountID: payload.AccountID, RefreshToken: input.RefreshToken,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		response.Send(ctx, 0, nil, apperrors.SessionUnavailable, "")
		return
	}
	if err != nil {
		response.Send(ctx, 0, nil, err, "revoke failed")
		return
	}
	ctx.Status(http.StatusNoContent)
}

// Me returns the verified account identity from the route authentication middleware.
func (c *AuthHandler) Me(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"account_id": middleware.AccountID(ctx)})
}
