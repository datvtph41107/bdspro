package auth

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/app/http/request"
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"github.com/datvtph41107/bdspro/app/http/resources"
	"github.com/datvtph41107/bdspro/app/http/responses"
	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Register(ctx *gin.Context) {
	var input authrequest.RegisterBody
	if !request.JSON(ctx, &input) {
		return
	}
	if !c.limiter.Identity(ctx, "register", input.Email, 10, 3, time.Hour) {
		return
	}
	account, err := auth.Register(ctx.Request.Context(), c.store, c.scheduler, input.Email, input.Password)
	responses.Send(ctx, http.StatusCreated, resources.NewUser(account), err, "registration failed")
}

func (c *Controller) Login(ctx *gin.Context) {
	var input authrequest.LoginBody
	if !request.JSON(ctx, &input) {
		return
	}
	if !c.limiter.Identity(ctx, "login", input.Email, 30, 10, time.Minute) {
		return
	}
	result, err := auth.Login(ctx.Request.Context(), c.store, c.maker, c.config,
		input.Email, input.Password, ctx.Request.UserAgent(), ctx.ClientIP())
	responses.Send(ctx, http.StatusOK, resources.NewLogin(result), err, "login failed")
}
