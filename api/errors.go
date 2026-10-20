package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// writeAPIError is the sole HTTP error serializer. Only explicit public
// messages reach the client; never pass err.Error() from service/pgx into it.
// Successful response schemas and 204 bodies remain unchanged.
func writeAPIError(ctx *gin.Context, status int, publicMessage string) {
	var code string
	switch status {
	case http.StatusBadRequest:
		code = "invalid_request"
	case http.StatusUnauthorized:
		code = "unauthorized"
	case http.StatusForbidden:
		code = "forbidden"
	case http.StatusNotFound:
		code = "not_found"
	case http.StatusMethodNotAllowed:
		code = "method_not_allowed"
	case http.StatusConflict:
		code = "conflict"
	case http.StatusRequestEntityTooLarge:
		code = "payload_too_large"
	case http.StatusUnsupportedMediaType:
		code = "unsupported_media_type"
	case http.StatusServiceUnavailable:
		code = "service_unavailable"
	default:
		code = "internal_error"
		status = http.StatusInternalServerError
		publicMessage = "internal server error"
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(status, gin.H{
		"error": gin.H{
			"code":       code,
			"message":    publicMessage,
			"request_id": ctx.GetString(requestIDKey),
		},
	})
}
