package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/hibiken/asynq"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type createUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type userResponse struct {
	AccountID int64     `json:"account_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	IsEmailVerified bool `json:"is_email_verified"`
}

func newUserResponse(account db.Account) userResponse {
	return userResponse{AccountID: account.ID, Email: account.Email, CreatedAt: account.CreatedAt, IsEmailVerified: account.IsEmailVerified}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	hash, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	account, err := server.store.RegisterTx(ctx, db.RegisterTxParams{
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: hash,
		AfterCreate: func(created db.Account) error {
			return server.taskDistributor.DistributeTaskSendVerifyEmail(
				ctx, &worker.PayloadSendVerifyEmail{AccountID: created.ID},
				asynq.MaxRetry(10), asynq.ProcessIn(10*time.Second), asynq.Queue(worker.QueueCritical),
			)
		},
	})
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	account, err := server.store.GetAccountByEmail(ctx, strings.TrimSpace(req.Email))
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	if err := util.CheckPassword(req.Password, account.PasswordHash); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		account.ID, server.config.AccessTokenDuration, token.TokenTypeAccess,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		account.ID, server.config.RefreshTokenDuration, token.TokenTypeRefresh,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		AccountID:    account.ID,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	ctx.JSON(http.StatusOK, loginUserResponse{
		SessionID:             session.ID.String(),
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(account),
	})
}
