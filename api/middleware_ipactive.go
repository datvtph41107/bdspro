package api

import (
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
)

// IPActive checks action-specific admission before entering a handler.
// It is not an authentication or authorization decision.
func (server *Server) IPActive(operation string, perIP, perIdentity int64, window time.Duration, identity func(*gin.Context) string) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        if !server.allowAuthRequest(ctx, operation, identity(ctx), perIP, perIdentity, window) {
            ctx.Abort()
            return
        }
        ctx.Next()
    }
}

func loginIdentity(ctx *gin.Context) string { return loginInput(ctx).Email }
func registerIdentity(ctx *gin.Context) string { return registerInput(ctx).Email }
func tokenIdentity(ctx *gin.Context) string { return tokenInput(ctx).RefreshToken }
func emailVerificationIdentity(ctx *gin.Context) string {
    return strconv.FormatInt(verifyEmailInput(ctx).EmailID, 10)
}
