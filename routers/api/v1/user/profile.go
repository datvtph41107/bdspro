package user

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func accountActionReady(ctx *appctx.APIContext) bool {
	ctx.Header("Cache-Control", "no-store")
	if ctx.Doer == nil {
		ctx.AuthenticationRequired()
		return false
	}
	return true
}

func writeAccountResult(ctx *appctx.APIContext, result any, err error) {
	if errors.Is(err, util.ErrUnauthenticated) {
		ctx.AuthenticationRequired()
		return
	}
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func GetProfile(ctx *appctx.APIContext) {
	if !accountActionReady(ctx) {
		return
	}
	result, err := authservice.GetProfile(ctx, ctx.Doer.AccountID)
	writeAccountResult(ctx, result, err)
}

func UpdateProfile(ctx *appctx.APIContext) {
	if !accountActionReady(ctx) {
		return
	}
	form := web.GetForm[*structs.UpdateProfileOption](ctx)
	result, err := authservice.UpdateProfile(ctx, *ctx.Doer, form.DisplayName, form.FirstName, form.LastName)
	writeAccountResult(ctx, result, err)
}

func GetSettings(ctx *appctx.APIContext) {
	if !accountActionReady(ctx) {
		return
	}
	result, err := authservice.GetAccountSettings(ctx, ctx.Doer.AccountID)
	writeAccountResult(ctx, result, err)
}

func UpdateSettings(ctx *appctx.APIContext) {
	if !accountActionReady(ctx) {
		return
	}
	form := web.GetForm[*structs.UpdateAccountSettingsOption](ctx)
	result, err := authservice.UpdateSettings(ctx, *ctx.Doer, form.Locale, form.Timezone, form.Currency)
	writeAccountResult(ctx, result, err)
}

func GetCredentials(ctx *appctx.APIContext) {
	if !accountActionReady(ctx) {
		return
	}
	result, err := authservice.GetCredentials(ctx, ctx.Doer.AccountID)
	writeAccountResult(ctx, result, err)
}

func ListConnections(ctx *appctx.APIContext) {
	if !accountActionReady(ctx) {
		return
	}
	items, err := authservice.ListConnections(ctx, ctx.Doer.AccountID)
	writeAccountResult(ctx, map[string]any{"items": items, "limit": 50}, err)
}

func UpdateUsername(ctx *appctx.APIContext) {
	if !accountActionReady(ctx) {
		return
	}
	form := web.GetForm[*structs.UpdateUsernameRequest](ctx)
	result, err := authservice.UpdateUsername(ctx, *ctx.Doer, form.Username)
	writeAccountResult(ctx, result, err)
}
