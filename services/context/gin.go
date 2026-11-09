package context

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/reqctx"

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
func (s ginRequestSource) WriteJSON(status int, value any) { s.ctx.JSON(status, value) }
func (s ginRequestSource) WriteStatus(status int)          { s.ctx.Status(status) }
func (s ginRequestSource) AbortJSON(status int, value any) { s.ctx.AbortWithStatusJSON(status, value) }
func (s ginRequestSource) SetHeader(name, value string)    { s.ctx.Header(name, value) }
func (s ginRequestSource) SetErrorCode(code string)        { s.ctx.Set(ErrorCodeKey, code) }
func (s ginRequestSource) IsAborted() bool                 { return s.ctx.IsAborted() }
func (s ginRequestSource) WrittenStatus() int {
	if !s.ctx.Writer.Written() {
		return 0
	}
	return s.ctx.Writer.Status()
}

// GetAPIContext returns one initialized context per request. Resource
// assignment, form binding and HTTP action share that exact instance.
func GetAPIContext(c *gin.Context) *APIContext {
	if value, ok := c.Get(apiContextKey); ok {
		if api, ok := value.(*APIContext); ok && api != nil {
			return api
		}
		panic("context: APIContext type mismatch")
	}
	api := &APIContext{Base: NewBaseContext(ginRequestSource{ctx: c}), Org: &APIOrganization{}}
	c.Set(apiContextKey, api)
	return api
}

func APIContexter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Request-local data and cleanup belong to modules/reqctx.
		// Preserve an existing scope if an outer ingress installed one.
		if reqctx.FromContext(c.Request.Context()) != nil {
			GetAPIContext(c)
			c.Next()
			return
		}
		scoped, finished := reqctx.NewRequestContext(c.Request.Context())
		c.Request = c.Request.WithContext(scoped)
		defer finished()
		GetAPIContext(c)
		c.Next()
	}
}

func Wrap(handler APIHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(GetAPIContext(c))
	}
}
