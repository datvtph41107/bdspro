package api

import (
	"github.com/datvtph41107/bdspro/doc/statik"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Swagger UI is a development-only endpoint. The OpenAPI JSON is embedded.
func (server *Server) registerSwagger(router *gin.Engine) {
	if server.config.Environment != "development" {
		return
	}
	router.GET("/swagger/*filepath", gin.WrapH(http.StripPrefix("/swagger/", http.FileServer(http.FS(statik.Files())))))
}
