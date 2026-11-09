package user

import (
	"net/http"

	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	profile_service "github.com/datvtph41107/bdspro/services/profile"
)

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
