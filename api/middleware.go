package api

import (
	"net/http"
	"strings"

	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

const authorizationPayloadKey = "authorization_payload"

// Authenticate verifies BDSPro access tokens. Resource-specific Authorize rules
// remain in SQL so ownership is checked atomically with the write.
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
		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}
