package response

import (
	"errors"
	"net/http"

	apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Invalid writes an invalid HTTP input response. Body-size errors retain 413.
func Invalid(c *gin.Context, err error) {
	var size *http.MaxBytesError
	if errors.As(err, &size) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}

// Error translates an application error into a public HTTP error response.
// Unknown errors are logged and never exposed to the client.
func Error(c *gin.Context, err error) {
	if public, ok := apperrors.Public(err); ok {
		c.JSON(public.Status, gin.H{"error": public.Message})
		return
	}
	log.Ctx(c.Request.Context()).Error().Err(err).Msg("http.application_error")
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
