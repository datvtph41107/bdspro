package response

import (
	"errors"
	"fmt"
	"net/http"

	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const ErrorCodeKey = "public_error_code"

// Reject emits one public, stable error contract; underlying errors are never serialized.
func Reject(c *gin.Context, status int, code, message string) {
	c.Set(ErrorCodeKey, code)
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}

func Invalid(c *gin.Context, err error) {
	var size *http.MaxBytesError
	if errors.As(err, &size) {
		Reject(c, http.StatusRequestEntityTooLarge, "request_body_too_large", "request body too large")
		return
	}
	Reject(c, http.StatusBadRequest, "invalid_request", "invalid request")
}

func Error(c *gin.Context, err error) {
	if public, ok := classifyApplicationError(err); ok {
		Reject(c, public.Status, public.Code, public.Message)
		return
	}
	// Do not serialize or log arbitrary err.Error(): driver and provider errors
	// may contain credentials, tokens, URLs or other sensitive values.
	event := log.Ctx(c.Request.Context()).Error().
		Str("error_type", fmt.Sprintf("%T", err))
	if pgErr := apperrors.Postgres(err); pgErr != nil {
		event = event.Str("postgres_code", pgErr.Code)
	}
	event.Msg("http.application_error")
	Reject(c, http.StatusInternalServerError, "internal_error", "internal server error")
}
