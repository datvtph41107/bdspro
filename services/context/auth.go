package context

import (
	"strings"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/gin-gonic/gin"
)

const AuthorizationPayloadKey = "authorization_payload"

// Authenticate verifies only the first-party access JWT signature and claims.
// Protected first-party routes then invoke RequireSession against PostgreSQL.
// Resource ownership stays in domain operations.
func Authenticate(maker authcrypto.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parts := strings.Fields(ctx.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			BearerRequired(ctx)
			return
		}
		payload, err := maker.VerifyToken(parts[1], authcrypto.TokenTypeAccess)
		if err != nil {
			InvalidAccessToken(ctx)
			return
		}
		ctx.Set(AuthorizationPayloadKey, payload)
		ctx.Next()
	}
}

func AccountID(ctx *gin.Context) int64 {
	return ctx.MustGet(AuthorizationPayloadKey).(*authcrypto.Payload).AccountID
}

// AuthorizationPayload returns only a verified access JWT payload installed
// by Authenticate. It does not look up current account or session state.
func AuthorizationPayload(c *gin.Context) (*authcrypto.Payload, bool) {
	v, ok := c.Get(AuthorizationPayloadKey)
	if !ok {
		return nil, false
	}
	p, ok := v.(*authcrypto.Payload)
	return p, ok && p != nil && p.AccountID > 0
}
