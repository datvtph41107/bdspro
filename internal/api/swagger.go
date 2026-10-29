package api

import (
	"net/http"

	_ "github.com/datvtph41107/bdspro/doc/statik"

	"github.com/gin-gonic/gin"
	"github.com/rakyll/statik/fs"
	"github.com/rs/zerolog/log"
)

func (server *Server) registerSwagger(router *gin.Engine) {
	if server.config.Environment != "development" {
		return
	}

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create statik fs")
	}

	swaggerHandler := http.StripPrefix(
		"/swagger/",
		http.FileServer(statikFS),
	)

	router.GET(
		"/swagger/*filepath",
		gin.WrapH(swaggerHandler),
	)
}
