package middleware

import (
	"net/http"
	"strings"

	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/gin-gonic/gin"
 "github.com/datvtph41107/bdspro/internal/api/response"
)

const AuthorizationPayloadKey = "authorization_payload"

// Authenticate supplies a verified AccountID to protected controller actions.
// Resource ownership remains with SQL predicates/transactions.
func Authenticate(maker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parts := strings.Fields(ctx.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Reject(ctx, http.StatusUnauthorized, "unauthenticated", "missing or invalid bearer token")
			return
		}
		payload, err := maker.VerifyToken(parts[1], token.TokenTypeAccess)
		if err != nil {
			response.Reject(ctx, http.StatusUnauthorized, "invalid_access_token", "invalid access token")
			return
		}
		ctx.Set(AuthorizationPayloadKey, payload)
		ctx.Next()
	}
}

func AccountID(ctx *gin.Context) int64 {
	return ctx.MustGet(AuthorizationPayloadKey).(*token.Payload).AccountID
}
