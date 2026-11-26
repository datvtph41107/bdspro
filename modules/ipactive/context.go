package ipactive

import "context"

type middlewareKey struct{}

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

func LookupMiddleware(ctx context.Context) (*Middleware, bool) {
	if ctx == nil {
		return nil, false
	}
	middleware, ok := ctx.Value(middlewareKey{}).(*Middleware)
	return middleware, ok && middleware != nil
}
