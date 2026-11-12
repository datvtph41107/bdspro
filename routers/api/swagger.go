package api

import (
	"io/fs"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/templates"
)

// RegisterSwagger keeps documentation routes in the same declaration
// path as application routes; Gin remains the only route matcher.
func RegisterSwagger(router *web.Router, environment string) {
	if environment != "development" {
		return
	}
	swaggerFS, err := fs.Sub(templates.Swagger, "swagger")
	if err != nil {
		panic(err)
	}
	fileServer := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS)))
	router.Get("/swagger/*filepath", func(w http.ResponseWriter, req *http.Request) {
		if req.PathValue("filepath") == "/index.html" {
			index, readErr := fs.ReadFile(swaggerFS, "index.html")
			if readErr != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(index)
			return
		}
		fileServer.ServeHTTP(w, req)
	})
}
