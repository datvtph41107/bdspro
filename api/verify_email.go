package api

import (
	"errors"
	"net/http"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type verifyEmailRequest struct {
	EmailID    int64  `form:"email_id" binding:"required,min=1"`
	SecretCode string `form:"secret_code" binding:"required"`
}

func (server *Server) verifyEmail(ctx *gin.Context) {
	var req verifyEmailRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification link"})
		return
	}
	_, err := server.store.VerifyEmailTx(ctx, db.ConsumeVerifyEmailParams{
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
