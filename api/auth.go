package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
)

// bindJSON preserves the legacy size, unknown-field and single-JSON-document
// checks while Gin performs binding and request validation.
func bindJSON(ctx *gin.Context, req any, limit int64) bool {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, limit)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil || !json.Valid(body) {
		return false
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
	return ctx.ShouldBindJSON(req) == nil
}

func replyJSON(ctx *gin.Context, status int, value any) {
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(status, value)
}

func (server *Server) cookieName() string {
	if !server.development {
		return "__Host-bdspro_session"
	}
	return "bdspro_session"
}

func (server *Server) cookie(ctx *gin.Context, token string, age int) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name: server.cookieName(), Value: token, Path: "/",
		Secure: !server.development, HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: age,
	})
}

func (server *Server) currentAccount(ctx *gin.Context) (int64, bool) {
	if server.authentication == nil {
		ctx.String(http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	cookie, err := ctx.Request.Cookie(server.cookieName())
	if err != nil {
		ctx.String(http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	id, err := server.authentication.AccountFromToken(ctx.Request.Context(), cookie.Value)
	if err != nil || id <= 0 {
		ctx.String(http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	return id, true
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

func (server *Server) verifyEmail(ctx *gin.Context) {
	var req verifyEmailRequest
	if !bindJSON(ctx, &req, 8192) {
		ctx.String(http.StatusBadRequest, "invalid input")
		return
	}
	if server.authentication == nil {
		ctx.String(http.StatusServiceUnavailable, "verification unavailable")
		return
	}
	err := server.authentication.VerifyEmail(ctx.Request.Context(), strings.TrimSpace(req.Token))
	if errors.Is(err, auth.ErrInvalidToken) {
		ctx.String(http.StatusBadRequest, "invalid or expired token")
		return
	}
	if err != nil {
		ctx.String(http.StatusInternalServerError, "verification failed")
		return
	}
	ctx.Status(http.StatusNoContent)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (server *Server) login(ctx *gin.Context) {
	var req loginRequest
	if !bindJSON(ctx, &req, 8192) {
		ctx.String(http.StatusBadRequest, "invalid input")
		return
	}
	if server.authentication == nil {
		ctx.String(http.StatusServiceUnavailable, "login unavailable")
		return
	}
	result, err := server.authentication.Login(ctx.Request.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		ctx.String(http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, auth.ErrUnverified):
		ctx.String(http.StatusForbidden, "email verification required")
	case err != nil:
		ctx.String(http.StatusInternalServerError, "login failed")
	default:
		server.cookie(ctx, result.Token, int(time.Until(result.ExpiresAt).Seconds()))
		replyJSON(ctx, http.StatusOK, gin.H{"account_id": result.AccountID})
	}
}

func (server *Server) me(ctx *gin.Context) {
	accountID, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	replyJSON(ctx, http.StatusOK, gin.H{"account_id": accountID})
}

func (server *Server) logout(ctx *gin.Context) {
	cookie, err := ctx.Request.Cookie(server.cookieName())
	if err != nil {
		server.cookie(ctx, "", -1)
		ctx.Status(http.StatusNoContent)
		return
	}
	if server.authentication == nil {
		ctx.String(http.StatusServiceUnavailable, "logout unavailable")
		return
	}
	if err := server.authentication.Logout(ctx.Request.Context(), cookie.Value); err != nil {
		ctx.String(http.StatusServiceUnavailable, "logout unavailable")
		return
	}
	server.cookie(ctx, "", -1)
	ctx.Status(http.StatusNoContent)
}
