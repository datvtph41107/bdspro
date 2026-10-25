package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// positiveInt64Param parses a positive decimal route ID. The caller owns
// the resource name and the HTTP error response for an invalid value.
func positiveInt64Param(ctx *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param(name), 10, 64)
	return id, err == nil && id > 0
}
