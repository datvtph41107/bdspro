package auth

import (
    "errors"
    "net/http"
    "strconv"
    "time"

    apperrors "github.com/datvtph41107/bdspro/app/errors"
    "github.com/datvtph41107/bdspro/app/http/request"
    authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
    "github.com/datvtph41107/bdspro/app/http/responses"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5"
)

func (c *Controller) VerifyEmail(ctx *gin.Context) {
	var input authrequest.VerifyEmailQuery
	if !request.Query(ctx, &input, "invalid verification link") {
		return
	}
	if !c.limiter.Identity(ctx, "verify_email", strconv.FormatInt(input.EmailID, 10), 60, 10, time.Minute) {
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
