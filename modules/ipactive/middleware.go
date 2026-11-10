package ipactive

import "context"

// Middleware is an HTTP-independent admission mechanism.
// Route denial policy is owned by services/security, not by Gin callbacks.
type Middleware struct {
	limiter *Limiter
}

func NewMiddleware(limiter *Limiter) *Middleware {
	return &Middleware{limiter: limiter}
}

func (m *Middleware) Check(ctx context.Context, action Action, sourceIP string) (Event, error) {
	if m == nil || m.limiter == nil {
		return Event{Action: action}, ErrUnavailable
	}
	return m.limiter.Check(ctx, action, sourceIP)
}
