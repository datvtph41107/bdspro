package server

import (
	middleware "github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
)

const authorizationPayloadKey = middleware.AuthorizationPayloadKey

// Compatibility entrypoint for existing transport tests and callers.
func Authenticate(maker token.Maker) gin.HandlerFunc {
	return middleware.Authenticate(maker)
}
