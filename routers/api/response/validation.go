package response

import (
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

type FieldIssue = appctx.FieldIssue

func FieldErrors(c *gin.Context, issues map[string][]FieldIssue) {
	appctx.FieldErrors(c, issues)
}

func FieldError(c *gin.Context, field, code, message string) {
	appctx.FieldError(c, field, code, message)
}

func Validation(c *gin.Context, err error, request any) {
	appctx.Validation(c, err, request)
}
