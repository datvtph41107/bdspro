package response

import (
	"errors"
	"net/http"

	apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Send is the single JSON response boundary used by ordinary API handlers.
// status is the successful HTTP status; a status of 400 with err indicates a
// request-binding failure. A status of 0 with err indicates an application
// failure. Existing public wire format {"error":"..."} is preserved.
func Send(ctx *gin.Context, status int, data any, err error, internalMessage string) {
	if err == nil {
		if status == http.StatusNoContent {
			ctx.Status(status)
			return
		}
		ctx.JSON(status, data)
		return
	}

	if status == http.StatusBadRequest {
		var size *http.MaxBytesError
		if errors.As(err, &size) {
			ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		if internalMessage == "" {
			internalMessage = "invalid request"
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": internalMessage})
		return
	}

	if public, ok := apperrors.Public(err); ok {
		ctx.JSON(public.Status, gin.H{"error": public.Message})
		return
	}
	log.Ctx(ctx.Request.Context()).Error().Err(err).Msg("http.application_error")
	if internalMessage == "" {
		internalMessage = "internal server error"
	}
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": internalMessage})
}
