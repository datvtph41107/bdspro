package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
)

type phoneStartRequest struct {
	Phone string `json:"phone" binding:"required"`
}
type phoneVerifyRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code string `json:"code" binding:"required"`
}

// startPhoneVerification is authenticated; OTP initiation does not confer
// authentication or Organization permissions by itself.
func (server *Server) startPhoneVerification(ctx *gin.Context) {
	var req phoneStartRequest
	if !bindJSON(ctx,&req,8192) {return}
	id,ok:=server.currentAccount(ctx)
	if !ok{return}
	if server.authentication==nil {
		writeAPIError(ctx,http.StatusServiceUnavailable,"SMS unavailable")
		return
	}
	err:=server.authentication.StartPhoneVerification(ctx.Request.Context(),id,strings.TrimSpace(req.Phone))
	switch{
	case errors.Is(err,auth.ErrPhoneInvalid):
		writeAPIError(ctx,http.StatusBadRequest,"invalid phone")
	case errors.Is(err,auth.ErrPhoneThrottle):
		ctx.Header("Retry-After","60")
		writeAPIError(ctx,http.StatusTooManyRequests,"verification temporarily limited")
	case errors.Is(err,auth.ErrSMSUnavailable):
		writeAPIError(ctx,http.StatusServiceUnavailable,"SMS unavailable")
	case errors.Is(err,auth.ErrUnauthorized):
		writeAPIError(ctx,http.StatusUnauthorized,"unauthorized")
	case err!=nil:
		writeAPIError(ctx,http.StatusInternalServerError,"phone verification unavailable")
	default:
		ctx.Status(http.StatusAccepted)
	}
}

func (server *Server) verifyPhoneCode(ctx *gin.Context) {
	var req phoneVerifyRequest
	if !bindJSON(ctx,&req,8192) {return}
	id,ok:=server.currentAccount(ctx)
	if !ok{return}
	if server.authentication==nil {
		writeAPIError(ctx,http.StatusServiceUnavailable,"verification unavailable")
		return
	}
	err:=server.authentication.VerifyPhoneCode(ctx.Request.Context(),id,strings.TrimSpace(req.Phone),req.Code)
	switch{
	case errors.Is(err,auth.ErrPhoneInvalid),errors.Is(err,auth.ErrPhoneCode):
		writeAPIError(ctx,http.StatusBadRequest,"invalid or expired phone verification code")
	case errors.Is(err,auth.ErrPhoneInUse):
		writeAPIError(ctx,http.StatusConflict,"phone unavailable")
	case errors.Is(err,auth.ErrSMSUnavailable):
		writeAPIError(ctx,http.StatusServiceUnavailable,"verification unavailable")
	case errors.Is(err,auth.ErrUnauthorized):
		writeAPIError(ctx,http.StatusUnauthorized,"unauthorized")
	case err!=nil:
		writeAPIError(ctx,http.StatusInternalServerError,"verification unavailable")
	default:
		ctx.Status(http.StatusNoContent)
	}
}
