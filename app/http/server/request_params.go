package server

import (
    "github.com/datvtph41107/bdspro/app/http/binding"
    "github.com/gin-gonic/gin"
)

func positiveInt64Param(ctx *gin.Context, name string) (int64, bool) {
    return binding.PositiveInt64Param(ctx, name)
}
