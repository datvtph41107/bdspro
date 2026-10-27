package responses

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// InvalidInput maps Gin's normal validation errors and oversized bodies
// without sending raw parser details to the client.
func InvalidInput(ctx *gin.Context, err error, message string) {
	var size *http.MaxBytesError
	if errors.As(err, &size) {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
		return
	}
	ctx.JSON(http.StatusBadRequest, gin.H{"error": message})
}
