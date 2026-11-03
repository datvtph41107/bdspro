package api

import (
	"io/fs"
	"net/http"

	"github.com/datvtph41107/bdspro/templates"
	"github.com/gin-gonic/gin"
)

// Swagger is local documentation; public runtime routes are owned by Gin.
func RegisterSwagger(router *gin.Engine, environment string) {
	if environment != "development" {
		return
	}
	swaggerFS, err := fs.Sub(templates.Swagger, "swagger")
	if err != nil {
		panic(err)
	}
	// net/http FileServer redirects explicit index.html to the directory.
	// Keep the reviewed URL /swagger/index.html directly usable while retaining
	// Go's safe file handling for every other static asset.
	fileServer := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS)))
	router.GET("/swagger/*filepath", func(c *gin.Context) {
		if c.Param("filepath") == "/index.html" {
			index, readErr := fs.ReadFile(swaggerFS, "index.html")
			if readErr != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", index)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
