package binding

import (
    "strconv"
    "github.com/gin-gonic/gin"
)

// PositiveInt64Param is a transport-only parser. It does not grant resource access.
func PositiveInt64Param(ctx *gin.Context, name string) (int64, bool) {
    id, err := strconv.ParseInt(ctx.Param(name), 10, 64)
    if err != nil || id <= 0 { return 0, false }
    return id, true
}
