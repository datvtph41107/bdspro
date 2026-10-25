package api

import (
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

// authenticatedAccountID reads the identity verified by Authenticate.
// It must only be called from routes protected by Authenticate. A missing
// payload is a route contract violation, not an anonymous/zero account.
func authenticatedAccountID(ctx *gin.Context) int64 {
	return ctx.MustGet(authorizationPayloadKey).(*token.Payload).AccountID
}
