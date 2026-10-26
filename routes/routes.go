package routes

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Handlers is the one-time HTTP composition input. It contains bound Gin
// handlers, not a registry, reflection-based controller resolver or global state.
type Handlers struct {
	Register, Login, VerifyEmail                                                     gin.HandlerFunc
	Renew, Revoke, Me                                                                gin.HandlerFunc
	CreateOrganization, ListOrganizations, AddOrganizationMember, UpdateOrganization gin.HandlerFunc
	PreRegister, PreLogin, PreVerifyEmail, PreToken, SufLogin, Authenticate          gin.HandlerFunc
	RegisterIdentity, LoginIdentity, VerifyEmailIdentity, TokenIdentity              func(*gin.Context) string
	Source                                                                           func(string, int64, time.Duration) gin.HandlerFunc
	Identity                                                                         func(string, int64, int64, time.Duration, func(*gin.Context) string) gin.HandlerFunc
}

func Mount(route *gin.Engine, h Handlers) {
	mountAuth(route, h)
	mountAPI(route, h)
}
