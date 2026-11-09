package context

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ginRequestSource is the single framework adapter into transport-neutral Base.
// It reads the current Request pointer, not a snapshot that becomes stale when
// RequireSession installs verified actor data.
type ginRequestSource struct {
	ctx *gin.Context
}

func (s ginRequestSource) CurrentRequest() *http.Request {
	return s.ctx.Request
}

func (s ginRequestSource) PathParam(name string) string {
	return s.ctx.Param(name)
}
