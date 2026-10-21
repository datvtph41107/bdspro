package api

import (
    "net/http"
    "time"

    "github.com/datvtph41107/bdspro/token"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5"
    "errors"
)

type renewAccessTokenRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

type renewAccessTokenResponse struct {
    AccessToken string `json:"access_token"`
    AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Server) renewAccessToken(ctx *gin.Context) {
    var req renewAccessTokenRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error":"invalid request"})
        return
    }
    payload, err := server.tokenMaker.VerifyToken(req.RefreshToken, token.TokenTypeRefresh)
    if err != nil {
        ctx.JSON(http.StatusUnauthorized, gin.H{"error":"invalid refresh token"})
        return
    }
    session, err := server.store.GetSession(ctx, payload.ID)
    if errors.Is(err, pgx.ErrNoRows) {
        ctx.JSON(http.StatusUnauthorized, gin.H{"error":"session unavailable"})
        return
    }
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error":"renew failed"})
        return
    }
    if session.IsBlocked || session.AccountID != payload.AccountID ||
        session.RefreshToken != req.RefreshToken || !time.Now().Before(session.ExpiresAt) {
        ctx.JSON(http.StatusUnauthorized, gin.H{"error":"session unavailable"})
        return
    }
    access, accessPayload, err := server.tokenMaker.CreateToken(
        payload.AccountID, server.config.AccessTokenDuration, token.TokenTypeAccess,
    )
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error":"renew failed"})
        return
    }
    ctx.JSON(http.StatusOK, renewAccessTokenResponse{
        AccessToken:access, AccessTokenExpiresAt:accessPayload.ExpiresAt.Time,
    })
}
