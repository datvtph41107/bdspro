package phonedelivery

import "context"

type serviceKey struct{}

// WithService binds the startup service to the existing function-based HTTP pipeline.
// Keep this boundary in routers; delivery policy itself uses Service methods.
func WithService(ctx context.Context, service *Service) context.Context {
	if service == nil {
		panic("phonedelivery: nil service")
	}
	return context.WithValue(ctx, serviceKey{}, service)
}

func FromContext(ctx context.Context) *Service {
	if ctx == nil {
		return nil
	}
	service, _ := ctx.Value(serviceKey{}).(*Service)
	return service
}
