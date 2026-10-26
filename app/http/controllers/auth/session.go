package auth

import (
    "errors"
    "net/http"
    "time"

    apperrors "github.com/datvtph41107/bdspro/app/errors"
    "github.com/datvtph41107/bdspro/app/http/request"
    authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
    "github.com/datvtph41107/bdspro/app/http/resources"
    "github.com/datvtph41107/bdspro/app/http/responses"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/token"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5"
)

func (c *Controller) Renew(ctx *gin.Context) {
	var input authrequest.TokenBody
	if !request.JSON(ctx, &input) {
		return
	}
	if !c.limiter.Identity(ctx, "refresh", input.RefreshToken, 60, 20, time.Minute) {
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
	if !request.JSON(ctx, &input) {
		return
	}
	if !c.limiter.Identity(ctx, "revoke", input.RefreshToken, 60, 20, time.Minute) {
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
