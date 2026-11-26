package auth

import "context"

type makerKey struct{}

func WithMaker(ctx context.Context, maker Maker) context.Context {
	if ctx == nil || maker == nil {
		panic("auth: missing token maker")
	}
	return context.WithValue(ctx, makerKey{}, maker)
}
func MakerFromContext(ctx context.Context) Maker {
	if ctx == nil {
		panic("auth: nil request context")
	}
	maker, ok := ctx.Value(makerKey{}).(Maker)
	if !ok || maker == nil {
		panic("auth: token maker missing from request")
	}
	return maker
}
