package middleware

import (
	"github.com/datvtph41107/bdspro/app/http/binding"
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"github.com/gin-gonic/gin"
)

func PrePhoneBind() gin.HandlerFunc {
	return binding.JSON[authrequest.PhoneBindBody](authrequest.PhoneKey)
}
func PrePhoneVerify() gin.HandlerFunc {
	return binding.JSON[authrequest.PhoneVerifyBody](authrequest.PhoneKey)
}
func PrePhoneLoginStart() gin.HandlerFunc {
	return binding.JSON[authrequest.PhoneStartBody](authrequest.PhoneKey)
}
func PhoneBindIdentity(ctx *gin.Context) string {
	return binding.Input[authrequest.PhoneBindBody](ctx, authrequest.PhoneKey).Phone
}
func PhoneVerifyIdentity(ctx *gin.Context) string {
	return binding.Input[authrequest.PhoneVerifyBody](ctx, authrequest.PhoneKey).Phone
}
func PhoneStartIdentity(ctx *gin.Context) string {
	return binding.Input[authrequest.PhoneStartBody](ctx, authrequest.PhoneKey).Phone
}
