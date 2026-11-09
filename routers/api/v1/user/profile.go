package user

import (
	"net/http"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/structs"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	profile_service "github.com/datvtph41107/bdspro/services/profile"
)

// ProfileHandler exposes only the verified caller's profile and preferences.
// The caller cannot override account_id or person_id in an HTTP body.
type ProfileHandler struct {
	profile *profile_service.Service
}

func NewProfileHandler(store db.Store) *ProfileHandler {
	return &ProfileHandler{profile: profile_service.NewService(store)}
}

func (h *ProfileHandler) Show(c *appctx.APIContext) {
	p, err := h.profile.Show(c, c.Doer.AccountID)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToAccountProfile(p))
}

func (h *ProfileHandler) Update(c *appctx.APIContext) {
	var req structs.ProfileUpdateRequest
	if err := c.BindJSON(&req); err != nil {
		c.APIInvalid(err)
		return
	}
	p, err := h.profile.Update(c, c.Doer.AccountID, c.Doer.SessionID, profile_service.ProfilePatch{
		DisplayName: req.DisplayName, FirstName: req.FirstName, LastName: req.LastName,
	})
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToAccountProfile(p))
}

func (h *ProfileHandler) Settings(c *appctx.APIContext) {
	p, err := h.profile.Show(c, c.Doer.AccountID)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToAccountSettings(p))
}

func (h *ProfileHandler) UpdateSettings(c *appctx.APIContext) {
	var req structs.SettingsUpdateRequest
	if err := c.BindJSON(&req); err != nil {
		c.APIInvalid(err)
		return
	}
	p, err := h.profile.UpdateSettings(c, c.Doer.AccountID, c.Doer.SessionID, profile_service.SettingsPatch{
		Locale: req.Locale, Timezone: req.Timezone, Currency: req.Currency,
	})
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToAccountSettings(p))
}
