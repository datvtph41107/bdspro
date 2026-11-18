package auth

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/web"
	token "github.com/datvtph41107/bdspro/modules/auth"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// Refresh exchanges a single-use opaque credential for a new session.
// Public route protection is provided by ipactive.RenewAccess before binding.
func Refresh(ctx *appctx.APIContext) {
	form := web.GetForm[*structs.RefreshSessionOption](ctx)

	result, err := authservice.RefreshSession(
		ctx,
		form.SessionID,
		form.RefreshToken,
		ctx.CurrentRequest().UserAgent(),
		ctx.ClientIP(),
		token.ProcessMaker(),
		setting.Security.AccessTokenDuration,
		setting.Security.RefreshTokenDuration,
	)
	if errors.Is(err, util.ErrUnauthenticated) {
		ctx.APIError(http.StatusUnauthorized, "invalid_refresh", "invalid or expired refresh credential")
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
