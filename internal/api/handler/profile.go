package handler

import (
	"net/http"

	"github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/datvtph41107/bdspro/internal/service"
	"github.com/gin-gonic/gin"
)

// ProfileHandler exposes the signed-in account; no account IDs are accepted
// from the request body or URL for self-service operations.
type ProfileHandler struct {
	profile *service.ProfileService
}

func NewProfileHandler(profile *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profile: profile}
}

func (h *ProfileHandler) Show(c *gin.Context) {
	profile, err := h.profile.Show(c.Request.Context(), middleware.AccountID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, newAccountProfileResponse(profile))
}

func (h *ProfileHandler) Update(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	profile, err := h.profile.Update(
		c.Request.Context(), middleware.AccountID(c),
		req.DisplayName, req.FirstName, req.LastName,
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, newAccountProfileResponse(profile))
}

func (h *ProfileHandler) Settings(c *gin.Context) {
	profile, err := h.profile.Show(c.Request.Context(), middleware.AccountID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, newAccountSettingsResponse(profile))
}

func (h *ProfileHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	profile, err := h.profile.UpdateSettings(
		c.Request.Context(), middleware.AccountID(c),
		req.Locale, req.Timezone, req.Currency,
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, newAccountSettingsResponse(profile))
}
