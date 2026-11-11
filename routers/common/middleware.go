package common

import (
	"fmt"
	"io"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

// ProtocolMiddlewares owns the order of global HTTP ingress behavior, as in
// Gitea's routers/common/middleware.go. Keep Gin as the only executor.
func ProtocolMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		RequestID(),
		AccessLogger(),
		Recovery(nil),
	}
}

// Recovery wraps the one public API panic policy. The optional writer allows
// isolated recovery tests to suppress Gin's panic diagnostics.
func Recovery(writer io.Writer) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(writer, recoverAPI)
}

// recoverAPI is the shared recovery policy for the single Gin router.
// Once an HTTP response is committed its status/body cannot be replaced by
// a JSON error. The panic is still logged and the remaining chain is aborted.
func recoverAPI(ctx *gin.Context, recovered any) {
	path := ctx.FullPath()
	if path == "" {
		path = "<unmatched>"
	}
	log.Ctx(ctx.Request.Context()).Error().
		Str("method", ctx.Request.Method).
		Str("path", path).
		Str("panic_type", fmt.Sprintf("%T", recovered)).
		Msg("http.panic")

	api := appctx.GetAPIContext(ctx)
	if api.Written() {
		ctx.Abort()
		return
	}
	api.APIError(http.StatusInternalServerError, "internal_error", "internal server error")
}
