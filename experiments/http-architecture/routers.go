// Package spike contains comparable route registrations, not a new router abstraction.
// Three router choices share the same application contract.
package spike

import (
	"net/http"

	"github.com/datvtph41107/bdspro-http-architecture-spike/contract"
	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
)

func Chi() http.Handler {
	r := chi.NewRouter()
	r.Use(contract.SecurityHeaders)
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", contract.Health)
		api.Post("/orgs/{org}/teams", func(w http.ResponseWriter, req *http.Request) {
			contract.CreateTeam(w, req, chi.URLParam(req, "org"))
		})
	})
	return r
}

func Gin() http.Handler {
	// Benchmark the production-appropriate Gin router mode, not debug logging.
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	})
	api := r.Group("/api/v1")
	api.GET("/health", func(c *gin.Context) {
		contract.Health(c.Writer, c.Request)
	})
	api.POST("/orgs/:org/teams", func(c *gin.Context) {
		contract.CreateTeam(c.Writer, c.Request, c.Param("org"))
	})
	return r
}

func ServeMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", contract.Health)
	mux.HandleFunc("POST /api/v1/orgs/{org}/teams", func(w http.ResponseWriter, req *http.Request) {
		contract.CreateTeam(w, req, req.PathValue("org"))
	})
	return contract.SecurityHeaders(mux)
}
