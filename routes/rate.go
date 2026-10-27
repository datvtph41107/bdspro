package routes

import (
	"time"

	"github.com/gin-gonic/gin"
)

// RateGuard exposes the single source-based request limiter at the route boundary.
type RateGuard interface {
	IPActiveSource(operation string, maxAttempts int64, window time.Duration) gin.HandlerFunc
}
