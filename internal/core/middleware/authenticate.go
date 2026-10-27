package middleware

import (
	"net/http"
	"strings"

	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/gin-gonic/gin"
)

const AuthorizationPayloadKey = "authorization_payload"

// Authenticate supplies a verified AccountID to protected controller actions.
// Resource ownership remains with SQL predicates/transactions.
func Authenticate(maker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parts := strings.Fields(ctx.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid bearer token"})
			return
		}
		payload, err := maker.VerifyToken(parts[1], token.TokenTypeAccess)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
			return
		}
		ctx.Set(AuthorizationPayloadKey, payload)
		ctx.Next()
	}
}

func AccountID(ctx *gin.Context) int64 {
	return ctx.MustGet(AuthorizationPayloadKey).(*token.Payload).AccountID
}
