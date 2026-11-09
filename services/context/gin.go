package context

import (
	"net/http"

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
		GetAPIContext(c)
		c.Next()
	}
}

func Wrap(handler APIHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(GetAPIContext(c))
	}
}
