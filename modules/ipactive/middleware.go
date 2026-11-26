package ipactive

import "context"

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

func (m *Middleware) CheckSubject(ctx context.Context, action Action, subjectID string) (Event, error) {
	if m == nil || m.limiter == nil {
		return Event{}, ErrUnavailable
	}
	return m.limiter.CheckSubject(ctx, action, subjectID)
}
