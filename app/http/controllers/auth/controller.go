package auth

import (
	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
)

// Controller is the HTTP entrypoint. It owns no database connection,
// SQL string, transaction runtime or queue implementation.
type Controller struct {
	store     db.Store
	scheduler auth.VerificationTaskScheduler
	maker     token.Maker
	config    util.Config
	limiter   *middleware.RateLimiter
}

func NewController(store db.Store, scheduler auth.VerificationTaskScheduler, maker token.Maker, config util.Config, limiter *middleware.RateLimiter) *Controller {
	return &Controller{store: store, scheduler: scheduler, maker: maker, config: config, limiter: limiter}
}
