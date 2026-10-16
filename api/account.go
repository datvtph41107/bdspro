package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
)

type createAccountRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// createAccount corresponds to SimpleBank's CreateAccount handler, but
// registration remains the identity-owned, multi-table transaction of BDSPro.
func (server *Server) createAccount(ctx *gin.Context) {
	if server.authentication == nil || server.authentication.S == nil {
		ctx.String(http.StatusServiceUnavailable, "registration unavailable")
		return
	}
	if strings.EqualFold(ctx.GetHeader("Sec-Fetch-Site"), "cross-site") {
		ctx.String(http.StatusForbidden, "cross-site request denied")
		return
	}
	origin := ctx.GetHeader("Origin")
	if origin != "" && (server.authentication.AllowedOrigin == "" ||
		origin != server.authentication.AllowedOrigin) {
		ctx.String(http.StatusForbidden, "origin not allowed")
		return
	}

	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 8192)
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.String(http.StatusBadRequest, "invalid input")
		return
	}

	registration, err := server.authentication.S.Register(
		ctx.Request.Context(), req.Email, req.Password,
	)
	switch {
	case errors.Is(err, auth.ErrInvalid):
		ctx.String(http.StatusBadRequest, "invalid input")
		return
	case errors.Is(err, auth.ErrEmailUsed):
		ctx.String(http.StatusConflict, "email unavailable")
		return
	case errors.Is(err, auth.ErrMailerUnavailable):
		ctx.String(http.StatusServiceUnavailable, "verification delivery unavailable; contact support")
		return
	case err != nil:
		ctx.String(http.StatusInternalServerError, "registration failed")
		return
	}

	result := gin.H{
		"account_id":            registration.AccountID,
		"verification_required": true,
	}
	if server.authentication.Development {
		result["verification_token"] = registration.VerificationToken
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusCreated, result)
}
