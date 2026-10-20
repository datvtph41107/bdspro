package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
)

// bindJSON validates the wire format before Gin's existing struct validation.
// Missing Content-Type remains accepted for legacy non-browser clients;
// explicit non-JSON media types are rejected. Failure writes the response.
func bindJSON(ctx *gin.Context, req any, limit int64) bool {
	if contentType := ctx.GetHeader("Content-Type"); contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil || mediaType != "application/json" {
			writeAPIError(ctx, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return false
		}
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, limit)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeAPIError(ctx, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeAPIError(ctx, http.StatusBadRequest, "invalid request body")
		}
		return false
	}
	// All existing request bodies are JSON objects, never scalars or arrays.
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' || !json.Valid(trimmed) {
		writeAPIError(ctx, http.StatusBadRequest, "invalid request body")
		return false
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err := ctx.ShouldBindJSON(req); err != nil {
		writeAPIError(ctx, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
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
	if id := ctx.GetInt64("bdspro_account_id"); id > 0 {
		return id, true
	}
	if server.authentication == nil {
		writeAPIError(ctx, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	cookie, err := ctx.Request.Cookie(server.cookieName())
	if err != nil {
		writeAPIError(ctx, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	id, err := server.authentication.AccountFromToken(ctx.Request.Context(), cookie.Value)
	if errors.Is(err, auth.ErrUnauthorized) || (err == nil && id <= 0) {
		writeAPIError(ctx, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	if err != nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "authentication unavailable")
		return 0, false
	}
	ctx.Set("bdspro_account_id", id)
	return id, true
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

func (server *Server) verifyEmail(ctx *gin.Context) {
	var req verifyEmailRequest
	if !bindJSON(ctx, &req, 8192) {
		return
	}
	if server.authentication == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "verification unavailable")
		return
	}
	err := server.authentication.VerifyEmail(ctx.Request.Context(), strings.TrimSpace(req.Token))
	if errors.Is(err, auth.ErrInvalidToken) {
		writeAPIError(ctx, http.StatusBadRequest, "invalid or expired token")
		return
	}
	if err != nil {
		writeAPIError(ctx, http.StatusInternalServerError, "verification failed")
		return
	}
	ctx.Status(http.StatusNoContent)
}

type resendVerificationRequest struct {
	Email string `json:"email" binding:"required"`
}

// Return the same response for unknown, already verified, and throttled addresses.
// Delivery/database failures are logged internally, never exposed as an enumeration oracle.
func (server *Server) resendVerification(ctx *gin.Context) {
	var req resendVerificationRequest
	if !bindJSON(ctx, &req, 8192) {
		return
	}
	if server.authentication == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "verification unavailable")
		return
	}
	err := server.authentication.ResendVerification(ctx.Request.Context(), req.Email)
	if errors.Is(err, auth.ErrInvalid) {
		writeAPIError(ctx, http.StatusBadRequest, "invalid input")
		return
	}
	if err != nil {
		// Backend errors may contain email addresses or provider diagnostics.
		server.logger.WarnContext(ctx.Request.Context(), "verification_resend_failed",
			"request_id", ctx.GetString("request_id"))
	}
	ctx.Status(http.StatusAccepted)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (server *Server) login(ctx *gin.Context) {
	var req loginRequest
	if !bindJSON(ctx, &req, 8192) {
		return
	}
	if server.authentication == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "login unavailable")
		return
	}
	result, err := server.authentication.Login(ctx.Request.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeAPIError(ctx, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, auth.ErrUnverified):
		writeAPIError(ctx, http.StatusForbidden, "email verification required")
	case err != nil:
		writeAPIError(ctx, http.StatusInternalServerError, "login failed")
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
		writeAPIError(ctx, http.StatusServiceUnavailable, "logout unavailable")
		return
	}
	if err := server.authentication.Logout(ctx.Request.Context(), cookie.Value); err != nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "logout unavailable")
		return
	}
	server.cookie(ctx, "", -1)
	ctx.Status(http.StatusNoContent)
}
