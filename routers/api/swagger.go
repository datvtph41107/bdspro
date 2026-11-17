package api

import (
	"io/fs"
	"net/http"

	"github.com/datvtph41107/bdspro/templates"
)

// SwaggerHandler returns the development-only Swagger HTTP handler.
// It has no dependency on the application's Chi route matcher.
// A nil handler means documentation is disabled in this environment.
func SwaggerHandler(environment string) http.Handler {
	if environment != "development" {
		return nil
	}
	swaggerFS, err := fs.Sub(templates.Swagger, "swagger")
	if err != nil {
		panic(err)
	}
	fileServer := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS)))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// FileServer redirects explicit index.html to its directory. Keep
		// the reviewed documentation URL working during matcher migration.
		if req.URL.Path == "/swagger/index.html" {
			index, err := fs.ReadFile(swaggerFS, "index.html")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
