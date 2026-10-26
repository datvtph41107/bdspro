package server

import (
    "errors"
    "net/http"
    "net/url"
    "time"

    authcontrollers "github.com/datvtph41107/bdspro/app/http/controllers/auth"
    "github.com/datvtph41107/bdspro/routes"
    "github.com/go-redis/redis/v8"
)

// EnableGoogleOAuth wires a configured auth controller into Gin routes.
// The OAuth state, callback and service logic live in controllers/auth.
func (server *Server) EnableGoogleOAuth(redisClient *redis.Client) error {
    if server.config.GoogleClientID == "" { return nil }
    if redisClient == nil { return errors.New("Google OAuth requires Redis") }
    redirect, err := url.Parse(server.config.GoogleRedirectURI)
    if err != nil { return err }
    controller := authcontrollers.NewGoogleController(
        server.store, server.tokenMaker, server.config, redisClient,
        &http.Client{Timeout:10*time.Second}, redirect.Scheme=="https",
    )
    routes.MountGoogle(server.router, routes.GoogleHandlers{
        StartLink:controller.StartLink, StartLogin:controller.StartLogin, Callback:controller.Callback,
        Authenticate:Authenticate(server.tokenMaker), Source:server.IPActiveSource,
    })
    return nil
}
