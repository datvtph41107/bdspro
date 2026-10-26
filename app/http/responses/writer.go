package responses

import (
    "net/http"

    apperrors "github.com/datvtph41107/bdspro/app/errors"
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog/log"
)

// Send writes a successful public resource or an error through one HTTP boundary.
// It preserves the existing {"error": "..."} wire format.
func Send(ctx *gin.Context, status int, data any, err error, internalMessage string) {
    if err != nil {
        Error(ctx, err, internalMessage)
        return
    }
    ctx.JSON(status, data)
}

func Error(ctx *gin.Context, err error, internalMessage string) {
    if public, ok := apperrors.Public(err); ok {
        ctx.JSON(public.Status, gin.H{"error":public.Message})
        return
    }
    log.Ctx(ctx.Request.Context()).Error().Err(err).Msg("http.application_error")
    if internalMessage == "" { internalMessage = "internal server error" }
    ctx.JSON(http.StatusInternalServerError, gin.H{"error":internalMessage})
}
