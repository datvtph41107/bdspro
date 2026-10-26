package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := uuid.NewString()
		ctx.Header("X-Request-ID", id)
		logger := log.With().Str("request_id", id).Logger()
		ctx.Request = ctx.Request.WithContext(logger.WithContext(ctx.Request.Context()))
		ctx.Next()
	}
}
func RequestLog() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		started := time.Now()
		ctx.Next()
		path := ctx.FullPath()
		if path == "" {
			path = "<unmatched>"
		}
		status := ctx.Writer.Status()
		event := log.Ctx(ctx.Request.Context()).Info()
		if status >= 500 {
			event = log.Ctx(ctx.Request.Context()).Error()
		}
		event.Str("method", ctx.Request.Method).Str("path", path).
			Int("status", status).Dur("duration", time.Since(started)).Msg("http.request")
	}
}
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(ctx *gin.Context, recovered any) {
		path := ctx.FullPath()
		if path == "" {
			path = "<unmatched>"
		}
		log.Ctx(ctx.Request.Context()).Error().Str("method", ctx.Request.Method).
			Str("path", path).Str("panic_type", fmt.Sprintf("%T", recovered)).Msg("http.panic")
		ctx.AbortWithStatus(http.StatusInternalServerError)
	})
}
