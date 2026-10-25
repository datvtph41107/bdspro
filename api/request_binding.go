package api

import "github.com/gin-gonic/gin"

// preJSON centralizes the repeated HTTP JSON admission contract.
// Named Pre* wrappers remain the readable entrypoints in route registration.
// A failed bind must abort before the business handler or identity limiter.
func preJSON[T any](key string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input T
		if err := ctx.ShouldBindJSON(&input); err != nil {
			invalidJSON(ctx, err)
			ctx.Abort()
			return
		}
		ctx.Set(key, input)
		ctx.Next()
	}
}
