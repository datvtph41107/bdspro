package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/gin-gonic/gin"
)

const maxRequestBodyBytes int64 = 1 << 20

// RequestIngress caps incoming JSON request bodies before controllers parse them.
func RequestIngress() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.ContentLength > maxRequestBodyBytes {
			response.Fail(ctx, response.RequestBodyTooLarge)
			return
		}
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxRequestBodyBytes)
		ctx.Next()
	}
}

func hashedRateKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
