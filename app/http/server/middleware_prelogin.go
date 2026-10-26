package server

import (
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	loginRequestContextKey       = authrequest.LoginKey
	registerRequestContextKey    = authrequest.RegisterKey
	tokenRequestContextKey       = authrequest.TokenKey
	verifyEmailRequestContextKey = authrequest.VerifyEmailKey
)

// PreLogin validates credentials syntax before the handler. Authentication
// remains in auth.Login and must precede session issuance.
func PreLogin() gin.HandlerFunc {
	return preJSON[loginUserRequest](loginRequestContextKey)
}

func PreRegister() gin.HandlerFunc {
	return preJSON[createUserRequest](registerRequestContextKey)
}

func PreToken() gin.HandlerFunc {
	return preJSON[refreshTokenRequest](tokenRequestContextKey)
}

// Verification links use query values rather than a JSON body.
func PreVerifyEmail() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input verifyEmailRequest
		if err := ctx.ShouldBindQuery(&input); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid verification link"})
			return
		}
		ctx.Set(verifyEmailRequestContextKey, input)
		ctx.Next()
	}
}

// The typed values below are set only by their route prevalidation middleware.
// Missing inputs are programming errors in route registration, not a fallback.
func loginInput(ctx *gin.Context) loginUserRequest {
	return ctx.MustGet(loginRequestContextKey).(loginUserRequest)
}

func registerInput(ctx *gin.Context) createUserRequest {
	return ctx.MustGet(registerRequestContextKey).(createUserRequest)
}

func tokenInput(ctx *gin.Context) refreshTokenRequest {
	return ctx.MustGet(tokenRequestContextKey).(refreshTokenRequest)
}

func verifyEmailInput(ctx *gin.Context) verifyEmailRequest {
	return ctx.MustGet(verifyEmailRequestContextKey).(verifyEmailRequest)
}
