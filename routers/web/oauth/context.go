package oauth

import "context"

type handlerKey struct{}

// Browser protocol state is app-local; no global handler registry.
func WithHandler(ctx context.Context, h *Handler) context.Context {
	if ctx==nil {panic("browser: nil request context")}
	if h==nil {return ctx}
	return context.WithValue(ctx,handlerKey{},h)
}
func FromContext(ctx context.Context) *Handler {
	if ctx==nil{return nil}
	h,_:=ctx.Value(handlerKey{}).(*Handler)
	return h
}
