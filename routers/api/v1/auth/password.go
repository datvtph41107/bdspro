package auth

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	token "github.com/datvtph41107/bdspro/modules/auth"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// LoginPassword admits a public login only after route-level IPActive
// throttling and validated JSON binding have succeeded.
func LoginPassword(ctx *appctx.APIContext) {
	form := web.GetForm[*structs.LoginPasswordOption](ctx)

	result, err := authservice.LoginPassword(
		ctx, form.Email, form.Password,
		ctx.CurrentRequest().UserAgent(), ctx.ClientIP(),
		token.ProcessMaker(),
		setting.Security.AccessTokenDuration,
		setting.Security.RefreshTokenDuration,
	)
	if errors.Is(err, authservice.ErrInvalidPasswordProof) {
		ctx.APIError(http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	ctx.JSON(http.StatusOK, result)
}
