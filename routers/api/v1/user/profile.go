package user

import (
	"net/http"

	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	profile_service "github.com/datvtph41107/bdspro/services/profile"
)

// GetAuthenticatedUser exposes the verified account; request-scoped identity
// and Store are resolved in services/profile, not injected into a handler.
func GetAuthenticatedUser(ctx *context.APIContext) {
	row, err := profile_service.GetAuthenticatedUser(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAccountProfile(row))
}

// UpdateProfile consumes a typed form published by modules/web at the
// route boundary. The account ID cannot be replaced by request fields.
func UpdateProfile(ctx *context.APIContext) {
	form := web.GetForm[*api.ProfileUpdateRequest](ctx)
	row, err := profile_service.UpdateCurrentProfile(ctx, profile_service.ProfilePatch{
		DisplayName: form.DisplayName,
		FirstName:   form.FirstName,
		LastName:    form.LastName,
	})
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAccountProfile(row))
}

// GetUserSettings reads the same account projection as GetAuthenticatedUser.
func GetUserSettings(ctx *context.APIContext) {
	row, err := profile_service.GetAuthenticatedUser(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAccountSettings(row))
}

// UpdateUserSettings preserves the transactional current-session check while
// sending only a profile settings patch to the service.
func UpdateUserSettings(ctx *context.APIContext) {
	form := web.GetForm[*api.SettingsUpdateRequest](ctx)
	row, err := profile_service.UpdateCurrentSettings(ctx, profile_service.SettingsPatch{
		Locale:   form.Locale,
		Timezone: form.Timezone,
		Currency: form.Currency,
	})
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAccountSettings(row))
}
