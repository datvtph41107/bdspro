package security

import "context"

type throttleKey struct{}

// WithThrottle binds the Redis admission capability to one server's requests.
// The limiter itself is safe for concurrent requests and is not copied.
func WithThrottle(ctx context.Context, throttle *AuthThrottle) context.Context {
	if ctx == nil || throttle == nil {
		panic("security: missing context or auth throttle")
	}
	return context.WithValue(ctx, throttleKey{}, throttle)
}

func ThrottleFromContext(ctx context.Context) *AuthThrottle {
	if ctx == nil {
		panic("security: nil context")
	}
	throttle, ok := ctx.Value(throttleKey{}).(*AuthThrottle)
	if !ok || throttle == nil {
		panic("security: auth throttle not initialized for request")
	}
	return throttle
}
