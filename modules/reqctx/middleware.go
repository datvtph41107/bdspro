package reqctx

import "net/http"

// RequestContextHandler is an outermost protocol middleware. As in Gitea,
// request-scoped state is established before route matching and closed after
// the last handler completes, whether it returned normally or panicked.
func RequestContextHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, finish := NewRequestContext(r.Context())
			defer finish()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
