package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func requestLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		slog.InfoContext(ctx.Request.Context(), "HTTP request",
			"method", ctx.Request.Method,
			"path", ctx.FullPath(),
			"status", ctx.Writer.Status(),
			"duration", time.Since(start),
		)
	}
}

func corsMiddleware(origins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if !allowed[origin] {
			ctx.Next()
			return
		}
		ctx.Header("Vary", "Origin")
		ctx.Header("Access-Control-Allow-Origin", origin)
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}
