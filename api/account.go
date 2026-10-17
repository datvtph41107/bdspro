package api

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
)

type createAccountRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// createAccount implements the SimpleBank handler pattern for BDSPro registration.
func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if !bindJSON(ctx, &req, 8192) {
		ctx.String(http.StatusBadRequest, "invalid input")
		return
	}
	if server.authentication == nil {
		ctx.String(http.StatusServiceUnavailable, "registration unavailable")
		return
	}
	registration, err := server.authentication.Register(
		ctx.Request.Context(), req.Email, req.Password,
	)
	switch {
	case errors.Is(err, auth.ErrInvalid):
		ctx.String(http.StatusBadRequest, "invalid input")
	case errors.Is(err, auth.ErrEmailUsed):
		ctx.String(http.StatusConflict, "email unavailable")
	case errors.Is(err, auth.ErrMailerUnavailable):
		ctx.String(http.StatusServiceUnavailable, "verification delivery unavailable; contact support")
	case err != nil:
		ctx.String(http.StatusInternalServerError, "registration failed")
	default:
		result := gin.H{
			"account_id": registration.AccountID, "verification_required": true,
		}
		if server.development {
			result["verification_token"] = registration.VerificationToken
		}
		replyJSON(ctx, http.StatusCreated, result)
	}
}
