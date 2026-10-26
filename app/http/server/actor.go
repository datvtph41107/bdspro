package server

import (
	middleware "github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/gin-gonic/gin"
)

func authenticatedAccountID(ctx *gin.Context) int64 {
	return middleware.AccountID(ctx)
}
