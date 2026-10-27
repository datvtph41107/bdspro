package routes

import (
	"time"

	"github.com/gin-gonic/gin"
)

// RateGuard is the narrow middleware dependency used by routes.
// It is implemented by the HTTP server; no handler registry is required.
type RateGuard interface {
	IPActiveSource(operation string, maxAttempts int64, window time.Duration) gin.HandlerFunc
	IPActive(operation string, perIP, perIdentity int64, window time.Duration, identity func(*gin.Context) string) gin.HandlerFunc
}
