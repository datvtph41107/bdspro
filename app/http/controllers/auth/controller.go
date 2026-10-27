package auth

import (
	"errors"
	"net/http"
	"time"

	apperrors "github.com/datvtph41107/bdspro/app/errors"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"github.com/datvtph41107/bdspro/app/http/resources"
	"github.com/datvtph41107/bdspro/app/http/responses"
	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Controller is the HTTP entrypoint. It owns no database connection,
// SQL string, transaction runtime or queue implementation.
type Controller struct {
	store     db.Store
	scheduler auth.VerificationTaskScheduler
	maker     token.Maker
	config    util.Config
}

func NewController(store db.Store, scheduler auth.VerificationTaskScheduler, maker token.Maker, config util.Config) *Controller {
	return &Controller{store: store, scheduler: scheduler, maker: maker, config: config}
}

func (c *Controller) Register(ctx *gin.Context) {
	var input authrequest.RegisterBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		responses.InvalidInput(ctx, err, "invalid request")
		return
	}
	account, err := auth.Register(ctx.Request.Context(), c.store, c.scheduler, input.Email, input.Password)
	responses.Send(ctx, http.StatusCreated, resources.NewUser(account), err, "registration failed")
}

func (c *Controller) Login(ctx *gin.Context) {
	var input authrequest.LoginBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		responses.InvalidInput(ctx, err, "invalid request")
		return
	}
	result, err := auth.Login(ctx.Request.Context(), c.store, c.maker, c.config,
		input.Email, input.Password, ctx.Request.UserAgent(), ctx.ClientIP())
	responses.Send(ctx, http.StatusOK, resources.NewLogin(result), err, "login failed")
}

func (c *Controller) VerifyEmail(ctx *gin.Context) {
	var input authrequest.VerifyEmailQuery
	if err := ctx.ShouldBindQuery(&input); err != nil {
		responses.InvalidInput(ctx, err, "invalid verification link")
		return
	}
	_, err := c.store.VerifyEmailTx(ctx.Request.Context(), db.ConsumeVerifyEmailParams{
		ID: input.EmailID, SecretCode: input.SecretCode,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		responses.Error(ctx, apperrors.InvalidVerificationLink, "")
		return
	}
	responses.Send(ctx, http.StatusOK, gin.H{"verified": true}, err, "verification failed")
}

func (c *Controller) Renew(ctx *gin.Context) {
	var input authrequest.TokenBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		responses.InvalidInput(ctx, err, "invalid request")
		return
	}
	payload, err := c.maker.VerifyToken(input.RefreshToken, token.TokenTypeRefresh)
	if err != nil {
		responses.Error(ctx, apperrors.InvalidRefreshToken, "")
		return
	}
	session, err := c.store.GetSession(ctx.Request.Context(), payload.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		responses.Error(ctx, apperrors.SessionUnavailable, "")
		return
	}
	if err != nil {
		responses.Error(ctx, err, "renew failed")
		return
	}
	if session.IsBlocked || session.AccountID != payload.AccountID ||
		session.RefreshToken != input.RefreshToken || !time.Now().Before(session.ExpiresAt) {
		responses.Error(ctx, apperrors.SessionUnavailable, "")
		return
	}
	access, tokenPayload, err := c.maker.CreateToken(
		payload.AccountID, c.config.AccessTokenDuration, token.TokenTypeAccess,
	)
	if err != nil {
		responses.Error(ctx, err, "renew failed")
		return
	}
	responses.Send(ctx, http.StatusOK, resources.RenewAccessToken{
		AccessToken: access, AccessTokenExpiresAt: tokenPayload.ExpiredAt,
	}, nil, "renew failed")
}

func (c *Controller) Revoke(ctx *gin.Context) {
	var input authrequest.TokenBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		responses.InvalidInput(ctx, err, "invalid request")
		return
	}
	payload, err := c.maker.VerifyToken(input.RefreshToken, token.TokenTypeRefresh)
	if err != nil {
		responses.Error(ctx, apperrors.InvalidRefreshToken, "")
		return
	}
	_, err = c.store.BlockSession(ctx.Request.Context(), db.BlockSessionParams{
		ID: payload.ID, AccountID: payload.AccountID, RefreshToken: input.RefreshToken,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		responses.Error(ctx, apperrors.SessionUnavailable, "")
		return
	}
	if err != nil {
		responses.Error(ctx, err, "revoke failed")
		return
	}
	ctx.Status(http.StatusNoContent)
}

// Me returns the verified account identity from the route authentication middleware.
func (c *Controller) Me(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"account_id": middleware.AccountID(ctx)})
}
