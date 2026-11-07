package middleware

import (
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

// AuthorizationPayloadKey remains a compatibility alias for existing callers.
// Principal state and verification are now owned by services/context.
const AuthorizationPayloadKey = appctx.AuthorizationPayloadKey

func Authenticate(maker authcrypto.Maker) gin.HandlerFunc {
	return appctx.Authenticate(maker)
}

func AccountID(c *gin.Context) int64 {
	return appctx.AccountID(c)
}
