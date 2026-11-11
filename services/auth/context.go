package auth

import "context"

// serviceKey is owned by services/auth. A Server binds its own AuthService at
// ingress; package-level router actions never receive an AuthHandler factory
// or read a mutable process-wide application container.
type serviceKey struct{}

// WithService binds a previously initialized service for this application's
// request. Separate Server instances retain independent dependency lifetimes.
func WithService(ctx context.Context, service *AuthService) context.Context {
	if ctx == nil || service == nil {
		panic("auth: missing request context or service")
	}
	return context.WithValue(ctx, serviceKey{}, service)
}

// FromContext returns the service owner installed by HTTP ingress.
// A missing service is a server construction bug, not an authentication error.
func FromContext(ctx context.Context) *AuthService {
	if ctx == nil {
		panic("auth: nil request context")
	}
	service, ok := ctx.Value(serviceKey{}).(*AuthService)
	if !ok || service == nil {
		panic("auth: service not initialized for request")
	}
	return service
}
