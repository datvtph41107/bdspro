package middleware

import (
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/app/http/binding"
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"github.com/gin-gonic/gin"
)

func PreLogin() gin.HandlerFunc {
	return binding.JSON[authrequest.LoginBody](authrequest.LoginKey)
}

func PreRegister() gin.HandlerFunc {
	return binding.JSON[authrequest.RegisterBody](authrequest.RegisterKey)
}

func PreToken() gin.HandlerFunc {
	return binding.JSON[authrequest.TokenBody](authrequest.TokenKey)
}

// Verification links keep their existing public error message.
func PreVerifyEmail() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input authrequest.VerifyEmailQuery
		if err := ctx.ShouldBindQuery(&input); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid verification link"})
			return
		}
		ctx.Set(authrequest.VerifyEmailKey, input)
		ctx.Next()
	}
}

func LoginIdentity(ctx *gin.Context) string {
	return binding.Input[authrequest.LoginBody](ctx, authrequest.LoginKey).Email
}

func RegisterIdentity(ctx *gin.Context) string {
	return binding.Input[authrequest.RegisterBody](ctx, authrequest.RegisterKey).Email
}

func TokenIdentity(ctx *gin.Context) string {
	return binding.Input[authrequest.TokenBody](ctx, authrequest.TokenKey).RefreshToken
}

func VerifyEmailIdentity(ctx *gin.Context) string {
	return strconv.FormatInt(binding.Input[authrequest.VerifyEmailQuery](ctx, authrequest.VerifyEmailKey).EmailID, 10)
}
