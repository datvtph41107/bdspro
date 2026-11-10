package context

import (
	"context"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web"

	"github.com/gin-gonic/gin"
)

const apiContextKey = "bdspro.api.context"

// ginRequestSource is the only Gin-to-Base request and response adapter.
// It retrieves the CURRENT request, including context installed by auth.
type ginRequestSource struct {
	ctx *gin.Context
}

func (s ginRequestSource) CurrentRequest() *http.Request { return s.ctx.Request }
func (s ginRequestSource) PathParam(name string) string  { return s.ctx.Param(name) }
func (s ginRequestSource) BindJSON(value any) error      { return s.ctx.ShouldBindJSON(value) }
func (s ginRequestSource) QueryDefault(name, fallback string) string {
	return s.ctx.DefaultQuery(name, fallback)
}

// JSON is encoded once in Base.JSON; Gin only writes the prepared bytes.
func (s ginRequestSource) WriteJSON(status int, encoded []byte) {
	s.ctx.Data(status, "application/json;charset=utf-8", encoded)
}
func (s ginRequestSource) WriteStatus(status int)       { s.ctx.Status(status) }
func (s ginRequestSource) Abort()                       { s.ctx.Abort() }
func (s ginRequestSource) SetHeader(name, value string) { s.ctx.Header(name, value) }
func (s ginRequestSource) SetErrorCode(code string)     { s.ctx.Set(ErrorCodeKey, code) }
func (s ginRequestSource) IsAborted() bool              { return s.ctx.IsAborted() }
func (s ginRequestSource) WrittenStatus() int {
	if !s.ctx.Writer.Written() {
		return 0
	}
	return s.ctx.Writer.Status()
}

// publishAPIContext attaches the same Gin-owned APIContext to the standard
// HTTP request, allowing modules/web to resolve typed actions without a
// router-specific accessor. It does not allocate another APIContext.
func publishAPIContext(c *gin.Context, api *APIContext) {
	if c.Request.Context().Value(apiContextRequestKey{}) != api {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), apiContextRequestKey{}, api))
	}
}

// GetAPIContext returns one initialized context per request. Resource
// assignment, form binding and HTTP action share that exact instance.
func GetAPIContext(c *gin.Context) *APIContext {
	if value, ok := c.Get(apiContextKey); ok {
		if api, ok := value.(*APIContext); ok && api != nil {
			publishAPIContext(c, api)
			return api
		}
		panic("context: APIContext type mismatch")
	}
	api := &APIContext{Base: NewBaseContext(ginRequestSource{ctx: c}), Org: &APIOrganization{}}
	c.Set(apiContextKey, api)
	publishAPIContext(c, api)
	return api
}

// APIContexter runs after route matching, as a native web.Router middleware.
// It creates exactly one APIContext for the matched Gin request, while the
// HTTP middleware chain owns continuation and request-scoped cleanup.
func APIContexter() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			c := web.GinContextFromRequest(req)
			if c == nil {
				panic("context: APIContexter requires a matched route")
			}
			if reqctx.FromContext(req.Context()) == nil {
				scoped, finished := reqctx.NewRequestContext(req.Context())
				defer finished()
				req = req.WithContext(scoped)
			}
			c.Request = req
			GetAPIContext(c)
			next.ServeHTTP(w, c.Request)
		})
	}
}

func Wrap(handler APIHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(GetAPIContext(c))
	}
}

// GinActions adapts existing Gin protocol handlers into typed API actions at
// registration time. It is the only compatibility bridge while authentication
// and OAuth endpoints still use Gin signatures. The request has one Gin source
// and one APIContext; no alternate middleware engine is introduced.
func GinActions(handlers ...gin.HandlerFunc) []any {
	actions := make([]any, len(handlers))
	for i, handler := range handlers {
		if handler == nil {
			panic("context: nil Gin HTTP handler")
		}
		h := handler
		actions[i] = func(ctx *APIContext) {
			if ctx == nil || ctx.Base == nil {
				panic("context: missing APIContext for Gin handler")
			}
			source, ok := ctx.Base.source.(ginRequestSource)
			if !ok {
				panic("context: Gin handler used outside Gin request")
			}
			h(source.ctx)
		}
	}
	return actions
}
