package middleware

import (
	"strings"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/gin-gonic/gin"
)

const AuthorizationPayloadKey = "authorization_payload"

// Authenticate supplies a verified AccountID to protected controller actions.
// Resource ownership remains with SQL predicates/transactions.
func Authenticate(maker authcrypto.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parts := strings.Fields(ctx.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.BearerRequired(ctx)
			return
		}
		payload, err := maker.VerifyToken(parts[1], authcrypto.TokenTypeAccess)
		if err != nil {
			response.InvalidAccessToken(ctx)
			return
		}
		ctx.Set(AuthorizationPayloadKey, payload)
		ctx.Next()
	}
}

func AccountID(ctx *gin.Context) int64 {
	return ctx.MustGet(AuthorizationPayloadKey).(*authcrypto.Payload).AccountID
}
