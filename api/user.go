package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type createUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type userResponse struct {
	AccountID       int64     `json:"account_id"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
	IsEmailVerified bool      `json:"is_email_verified"`
}

func newUserResponse(account db.Account) userResponse {
	return userResponse{
		AccountID: account.ID, Email: account.Email,
		CreatedAt: account.CreatedAt, IsEmailVerified: account.IsEmailVerified,
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		invalidJSON(ctx, err)
		return
	}
	if !server.allowAuthRequest(ctx, "register", req.Email, 10, 3, time.Hour) { return }
	account, err := auth.Register(ctx.Request.Context(), server.store, server.taskDistributor, req.Email, req.Password)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
			ctx.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}
	ctx.JSON(http.StatusCreated, newUserResponse(account))
}

type loginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginUserResponse struct {
	SessionID             string       `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		invalidJSON(ctx, err)
		return
	}
	if !server.allowAuthRequest(ctx, "login", req.Email, 30, 10, time.Minute) { return }
	result, err := auth.Login(ctx, server.store, server.tokenMaker, server.config,
		req.Email, req.Password, ctx.Request.UserAgent(), ctx.ClientIP())
	if errors.Is(err, auth.ErrInvalidCredentials) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	ctx.JSON(http.StatusOK, loginUserResponse{
		SessionID:   result.Session.ID.String(),
		AccessToken: result.AccessToken, AccessTokenExpiresAt: result.AccessExpiresAt,
		RefreshToken: result.RefreshToken, RefreshTokenExpiresAt: result.RefreshExpiresAt,
		User: newUserResponse(result.Account),
	})
}
