package context

import (
	"encoding/json"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

type Principal struct {
	AccountID int64
}

type APIContext struct {
	*Base
	Doer *Principal
}

type apiContextKey struct{}

// APIContexter initializes a request-scoped APIContext. The middleware must
// precede authentication and all API endpoints. It never creates app services.
func APIContexter() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			store := reqctx.MustFromContext(r.Context())
			base := NewBaseContext(w, r)
			ctx := &APIContext{Base: base}
			store.Set(apiContextKey{}, ctx)
			next.ServeHTTP(base.Resp, r)
		})
	}
}

func GetAPIContext(r *http.Request) *APIContext {
	if r == nil {
		panic("context: nil request")
	}
	store := reqctx.MustFromContext(r.Context())
	v, ok := store.Get(apiContextKey{})
	if !ok {
		panic("context: missing APIContext; install APIContexter")
	}
	ctx, ok := v.(*APIContext)
	if !ok {
		panic("context: unexpected APIContext type")
	}
	return ctx
}

func (ctx *APIContext) WrittenStatus() int { return ctx.Resp.WrittenStatus() }

func (ctx *APIContext) JSON(status int, value any) {
	if ctx.WrittenStatus() != 0 {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		ctx.APIError(http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	ctx.Resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.Resp.WriteHeader(status)
	_, _ = ctx.Resp.Write(data)
}

// APIError retains BDSPro's public {error,code} schema. Persistence and
// provider details must never be sent to clients.
func (ctx *APIContext) APIError(status int, code, message string) {
	if status == http.StatusInternalServerError {
		code, message = "internal_error", "internal server error"
	}
	ctx.JSON(status, struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}{Error: message, Code: code})
}

func (ctx *APIContext) APIErrorAuto(err error) {
	if err == nil {
		return
	}
	if status, code, message, ok := applicationError(err); ok {
		ctx.APIError(status, code, message)
		return
	}
	ctx.APIError(http.StatusInternalServerError, "internal_error", "internal server error")
}
