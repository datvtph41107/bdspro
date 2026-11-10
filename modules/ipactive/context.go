package ipactive

import "context"

type middlewareKey struct{}

// Binds admission capability with an unexported key to this server's request.
func WithMiddleware(ctx context.Context, middleware *Middleware) context.Context {
	if ctx == nil || middleware == nil {
		panic("ipactive: missing limiter")
	}
	return context.WithValue(ctx, middlewareKey{}, middleware)
}
func FromContext(ctx context.Context) *Middleware {
	if ctx == nil {
		panic("ipactive: nil request context")
	}
	middleware, ok := ctx.Value(middlewareKey{}).(*Middleware)
	if !ok || middleware == nil {
		panic("ipactive: limiter missing from request")
	}
	return middleware
}
