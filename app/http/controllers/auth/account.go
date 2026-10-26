package auth

import (
	"net/http"

	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/gin-gonic/gin"
)

// Me exposes only the identity established by authentication middleware.
func (c *Controller) Me(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"account_id": middleware.AccountID(ctx)})
}
