package api

import (
	"net/http"

	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Server serves HTTP requests for BDSPro.
type Server struct {
	store          *db.Store
	authentication *auth.Handler
	router         *gin.Engine
}

// NewServer creates a Gin server and sets up routing.
// The existing HTTP handlers remain the fallback until their Gin lectures.
func NewServer(store *db.Store, authentication *auth.Handler, legacy http.Handler) *Server {
	server := &Server{store: store, authentication: authentication}
	router := gin.Default()

	// Keep the existing strict JSON rule while adopting Gin's binding.
	binding.EnableDecoderDisallowUnknownFields = true

	router.POST("/auth/register", server.createAccount)
	router.GET("/health", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})

	if legacy != nil {
		router.NoRoute(gin.WrapH(legacy))
	}

	server.router = router
	return server
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// Handler returns the Gin router for the existing graceful shutdown server.
func (server *Server) Handler() http.Handler {
	return server.router
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
