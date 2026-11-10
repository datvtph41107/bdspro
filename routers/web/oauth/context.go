package oauth

import "context"

type browserKey struct{}

// Browser configuration is bound to the serving app's request and never
// obtained from a mutable global Handler registry.
func WithBrowser(ctx context.Context, b *Browser) context.Context {
	if ctx == nil {
		panic("browser: nil request context")
	}
	if b == nil {
		return ctx
	}
	return context.WithValue(ctx, browserKey{}, b)
}
func FromContext(ctx context.Context) *Browser {
	if ctx == nil {
		return nil
	}
	b, _ := ctx.Value(browserKey{}).(*Browser)
	return b
}
