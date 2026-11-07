package context

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxRequestBodyBytes int64 = 1 << 20

// RequestIngress caps incoming JSON request bodies before controllers parse them.
func RequestIngress() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.ContentLength > maxRequestBodyBytes {
			Reject(ctx, http.StatusRequestEntityTooLarge, "request_body_too_large", "request body too large")
			return
		}
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxRequestBodyBytes)
		ctx.Next()
	}
}
