package api

import (
	"io/fs"
	"net/http"

	assets "github.com/datvtph41107/bdspro/doc"
	"github.com/gin-gonic/gin"
)

// Swagger is local documentation; public runtime routes are owned by Gin.
func (server *Server) registerSwagger(router *gin.Engine) {
	if server.config.Environment != "development" {
		return
	}
	swaggerFS, err := fs.Sub(assets.Swagger, "swagger")
	if err != nil {
		panic(err)
	}
	router.GET("/swagger/*filepath", gin.WrapH(
		http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS))),
	))
}
