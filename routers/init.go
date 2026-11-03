// Package routers initializes the HTTP application without owning CLI lifecycle.
package routers

import (
    "os"

    "github.com/datvtph41107/bdspro/models/db"
    "github.com/datvtph41107/bdspro/modules/setting"
    "github.com/datvtph41107/bdspro/routers/api"
    "github.com/datvtph41107/bdspro/services/mail"
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
)

// InitWebInstalled initializes Gin policy and route-bound dependencies.
// The existing routers/api package owns route declarations and HTTP contracts.
func InitWebInstalled(cfg setting.Config, store db.Store, redisClient *redis.Client, sender mail.EmailSender) (*api.Server, error) {
    if os.Getenv(gin.EnvGinMode) == "" { gin.SetMode(gin.ReleaseMode) }
    return api.NewServer(cfg, store, redisClient, sender)
}
