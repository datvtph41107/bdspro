package api

import (
    authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
    "github.com/datvtph41107/bdspro/app/http/responses"
    "github.com/datvtph41107/bdspro/app/http/resources"
	"net/http"

	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/gin-gonic/gin"
)

type createUserRequest = authrequest.RegisterBody

type userResponse = resources.User

func newUserResponse(account db.Account) userResponse { return resources.NewUser(account) }

func (server *Server) createUser(ctx *gin.Context) {
	req := registerInput(ctx)
	account, err := auth.Register(ctx.Request.Context(), server.store, server.verificationScheduler, req.Email, req.Password)
	responses.Send(ctx, http.StatusCreated, newUserResponse(account), err, "registration failed")
}

type loginUserRequest = authrequest.LoginBody

type loginUserResponse = resources.Login

// newLoginUserResponse is the common public session representation for
// password, Google and phone OTP sign-in. Authentication stays provider-specific.
func newLoginUserResponse(result auth.LoginResult) loginUserResponse { return resources.NewLogin(result) }

func (server *Server) loginUser(ctx *gin.Context) {
	req := loginInput(ctx)
	result, err := auth.Login(ctx.Request.Context(), server.store, server.tokenMaker, server.config,
		req.Email, req.Password, ctx.Request.UserAgent(), ctx.ClientIP())
	responses.Send(ctx, http.StatusOK, newLoginUserResponse(result), err, "login failed")
}
