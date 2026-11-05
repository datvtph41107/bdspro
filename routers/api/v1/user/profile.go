package user

import (
	"net/http"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/datvtph41107/bdspro/services/convert"
	profile_service "github.com/datvtph41107/bdspro/services/profile"
	"github.com/gin-gonic/gin"
)

// ProfileHandler exposes only the verified caller's profile and preferences.
// The caller cannot override account_id or person_id in an HTTP body.
type ProfileHandler struct {
	profile *profile_service.Service
}

func NewProfileHandler(store db.Store) *ProfileHandler {
	return &ProfileHandler{profile: profile_service.NewService(store)}
}

func (h *ProfileHandler) Show(c *gin.Context) {
	p, err := h.profile.Show(c.Request.Context(), middleware.AccountID(c))
	if err != nil { response.Error(c, err); return }
	c.JSON(http.StatusOK, convert.ToAccountProfile(p))
}

func (h *ProfileHandler) Update(c *gin.Context) {
	var req structs.ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Invalid(c, err); return }
	p, err := h.profile.Update(c.Request.Context(), middleware.AccountID(c), profile_service.ProfilePatch{
		DisplayName: req.DisplayName, FirstName: req.FirstName, LastName: req.LastName,
	})
	if err != nil { response.Error(c, err); return }
	c.JSON(http.StatusOK, convert.ToAccountProfile(p))
}

func (h *ProfileHandler) Settings(c *gin.Context) {
	p, err := h.profile.Show(c.Request.Context(), middleware.AccountID(c))
	if err != nil { response.Error(c, err); return }
	c.JSON(http.StatusOK, convert.ToAccountSettings(p))
}

func (h *ProfileHandler) UpdateSettings(c *gin.Context) {
	var req structs.SettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Invalid(c, err); return }
	p, err := h.profile.UpdateSettings(c.Request.Context(), middleware.AccountID(c), profile_service.SettingsPatch{
		Locale: req.Locale, Timezone: req.Timezone, Currency: req.Currency,
	})
	if err != nil { response.Error(c, err); return }
	c.JSON(http.StatusOK, convert.ToAccountSettings(p))
}
