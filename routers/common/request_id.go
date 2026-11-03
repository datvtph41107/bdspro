package common

import (
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID must run before recovery, access logging and API admission checks.
func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := uuid.NewString()
		ctx.Header("X-Request-ID", requestID)
		requestLogger := log.With().Str("request_id", requestID).Logger()
		ctx.Request = ctx.Request.WithContext(requestLogger.WithContext(ctx.Request.Context()))
		ctx.Next()
	}
}
