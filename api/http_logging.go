package api

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

func newRequestID() (string, error) {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(entropy[:]), nil
}

// logRequest emits one event with only safe, predefined fields.
// Do not log raw URL/query, cookies, auth headers, client IP, or bodies.
func (server *Server) logRequest(ctx *gin.Context) {
	started := time.Now()
	requestID, err := newRequestID()
	if err != nil {
		ctx.AbortWithStatus(http.StatusServiceUnavailable)
	} else {
		// Never trust an inbound X-Request-ID for correlation authority.
		ctx.Set(requestIDKey, requestID)
		ctx.Header("X-Request-ID", requestID)
		ctx.Next()
	}

	route := ctx.FullPath()
	if route == "" {
		route = "unmatched"
	}
	status := ctx.Writer.Status()
	level := slog.LevelInfo
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	server.logger.LogAttrs(ctx.Request.Context(), level, "http_request",
		slog.String("request_id", requestID),
		slog.String("method", ctx.Request.Method),
		slog.String("route", route),
		slog.Int("status", status),
		slog.Float64("duration_ms", float64(time.Since(started).Microseconds())/1000),
		slog.Bool("panic_recovered", ctx.GetBool("panic_recovered")),
	)
}
