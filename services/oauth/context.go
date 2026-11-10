package oauth

import "context"

type serviceKey struct{}

type dependencies struct {
	service *Service
	proofs  *ConsentProofs
}

// A server attaches only its own initialized OAuth capabilities. No shared
// global Handler, token-store singleton or app-wide dependency container.
func WithService(ctx context.Context, service *Service, proofs *ConsentProofs) context.Context {
	if ctx == nil {
		panic("oauth: nil request context")
	}
	if service == nil {
		return ctx
	}
	return context.WithValue(ctx, serviceKey{}, dependencies{service: service, proofs: proofs})
}
func ServiceFromContext(ctx context.Context) *Service {
	if ctx == nil {
		return nil
	}
	owned, ok := ctx.Value(serviceKey{}).(dependencies)
	if !ok {
		return nil
	}
	return owned.service
}
func ProofsFromContext(ctx context.Context) *ConsentProofs {
	if ctx == nil {
		return nil
	}
	owned, ok := ctx.Value(serviceKey{}).(dependencies)
	if !ok {
		return nil
	}
	return owned.proofs
}
