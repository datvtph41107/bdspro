package api

import (
    authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"errors"
	"net/http"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type verifyEmailRequest = authrequest.VerifyEmailQuery

func (server *Server) verifyEmail(ctx *gin.Context) {
	req := verifyEmailInput(ctx)
	_, err := server.store.VerifyEmailTx(ctx.Request.Context(), db.ConsumeVerifyEmailParams{
		ID: req.EmailID, SecretCode: req.SecretCode,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification link"})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"verified": true})
}
