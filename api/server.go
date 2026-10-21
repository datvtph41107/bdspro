package api

import (
    "fmt"
    "github.com/gin-gonic/gin"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/token"
    "github.com/datvtph41107/bdspro/util"
)

type Server struct {
    config util.Config
    store db.Store
    tokenMaker token.Maker
    router *gin.Engine
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
    maker, err := token.NewJWTMaker(config.TokenSymmetricKey)
    if err != nil { return nil, fmt.Errorf("create token maker: %w", err) }
    server := &Server{config:config, store:store, tokenMaker:maker}
    server.setupRouter()
    return server, nil
}

func (server *Server) setupRouter() {
    router := gin.Default()
    router.POST("/users", server.createUser)
    router.POST("/users/login", server.loginUser)
    router.POST("/tokens/renew_access", server.renewAccessToken)

    authorized := router.Group("/").Use(authMiddleware(server.tokenMaker))
    authorized.GET("/users/me", server.me)
    server.router = router
}

func (server *Server) me(ctx *gin.Context) {
    payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
    ctx.JSON(200, gin.H{"account_id":payload.AccountID})
}

func (server *Server) Start(address string) error { return server.router.Run(address) }
