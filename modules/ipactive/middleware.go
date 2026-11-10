package ipactive

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Reject maps decisions to the application's existing HTTP error writer.
// The technical module does not import routers or duplicate JSON protocol.
type Reject func(*gin.Context, Event, error)

// Middleware packages Gin integration with the IPActive mechanism.
type Middleware struct {
	limiter *Limiter
	reject  Reject
}

func NewMiddleware(limiter *Limiter, reject Reject) *Middleware {
	return &Middleware{limiter: limiter, reject: reject}
}

// Check is the transport-independent admission mechanism. The trusted source
// IP is supplied by the HTTP transport and validated by Limiter.Check.
func (m *Middleware) Check(ctx context.Context, action Action, sourceIP string) (Event, error) {
	if m == nil || m.limiter == nil {
		return Event{Action: action}, ErrUnavailable
	}
	return m.limiter.Check(ctx, action, sourceIP)
}

func (m *Middleware) For(action Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m == nil || m.reject == nil {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		if m.limiter == nil {
			m.reject(c, Event{Action: action}, ErrUnavailable)
			c.Abort()
			return
		}
		event, err := m.limiter.Check(c.Request.Context(), action, c.ClientIP())
		if err != nil || !event.Allowed {
			m.reject(c, event, err)
			c.Abort()
			return
		}
		c.Next()
	}
}
