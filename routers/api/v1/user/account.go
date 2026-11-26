package user

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/util"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func GetCurrentUser(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	if ctx.Doer == nil {
		ctx.AuthenticationRequired()
		return
	}
	u, err := authservice.GetCurrentUser(ctx, ctx.Doer.AccountID)
	if errors.Is(err, util.ErrUnauthenticated) {
		ctx.AuthenticationRequired()
		return
	}
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, u)
}
