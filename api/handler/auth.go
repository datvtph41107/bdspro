package handler

import (
	"net/http"
	"strconv"
	"time"

	authrequest "github.com/datvtph41107/bdspro/api/handler/request/auth"
	"github.com/datvtph41107/bdspro/api/handler/resource"
	"github.com/datvtph41107/bdspro/api/service"
	"github.com/datvtph41107/bdspro/core/middleware"
	"github.com/datvtph41107/bdspro/core/response"
	"github.com/gin-gonic/gin"
)

// AuthHandler owns HTTP binding, identity-based admission and response
// projection. Authentication and session operations belong to AuthService.
type AuthHandler struct {
	auth    *service.AuthService
	limiter *middleware.IdentityLimiter
}

func NewAuthHandler(auth *service.AuthService, limiter *middleware.IdentityLimiter) *AuthHandler {
	return &AuthHandler{auth: auth, limiter: limiter}
}

func (h *AuthHandler) Register(ctx *gin.Context) {
	var input authrequest.RegisterBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if !h.limiter.Allow(ctx, "register", input.Email, 10, 3, time.Hour) {
		return
	}
	account, err := h.auth.Register(ctx.Request.Context(), input.Email, input.Password)
	response.Send(ctx, http.StatusCreated, resources.NewUser(account), err, "registration failed")
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	var input authrequest.LoginBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if !h.limiter.Allow(ctx, "login", input.Email, 30, 10, time.Minute) {
		return
	}
	result, err := h.auth.Login(ctx.Request.Context(),
		input.Email, input.Password, ctx.Request.UserAgent(), ctx.ClientIP())
	response.Send(ctx, http.StatusOK, resources.NewLogin(result), err, "login failed")
}

func (h *AuthHandler) VerifyEmail(ctx *gin.Context) {
	var input authrequest.VerifyEmailQuery
	if err := ctx.ShouldBindQuery(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid verification link")
		return
	}
	if !h.limiter.Allow(ctx, "verify_email", strconv.FormatInt(input.EmailID, 10), 60, 10, time.Minute) {
		return
	}
	err := h.auth.VerifyEmail(ctx.Request.Context(), input.EmailID, input.SecretCode)
	response.Send(ctx, http.StatusOK, gin.H{"verified": true}, err, "verification failed")
}

func (h *AuthHandler) Renew(ctx *gin.Context) {
	var input authrequest.TokenBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if !h.limiter.Allow(ctx, "refresh", input.RefreshToken, 60, 20, time.Minute) {
		return
	}
	result, err := h.auth.Renew(ctx.Request.Context(), input.RefreshToken)
	response.Send(ctx, http.StatusOK, resources.RenewAccessToken{
		AccessToken: result.AccessToken, AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	}, err, "renew failed")
}

func (h *AuthHandler) Revoke(ctx *gin.Context) {
	var input authrequest.TokenBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if !h.limiter.Allow(ctx, "revoke", input.RefreshToken, 60, 20, time.Minute) {
		return
	}
	err := h.auth.Revoke(ctx.Request.Context(), input.RefreshToken)
	response.Send(ctx, http.StatusNoContent, nil, err, "revoke failed")
}

// Me uses the account identity established by the route's JWT middleware.
func (h *AuthHandler) Me(ctx *gin.Context) {
	response.Send(ctx, http.StatusOK, gin.H{"account_id": middleware.AccountID(ctx)}, nil, "")
}
