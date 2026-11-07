package org

import (
	"net/http"
	"strconv"
	"strings"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
)

type Handler struct {
	store db.Store
}

func NewHandler(store db.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) List(c *appctx.APIContext) {
	organizations, err := organization_model.ListForAccount(c.Request.Context(), h.store, c.Doer.AccountID)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToOrganizationAccessList(organizations))
}

func (h *Handler) Get(c *appctx.APIContext) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	organization, err := organization_model.GetForAccount(c.Request.Context(), h.store, c.Doer.AccountID, id)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToOrganizationAccess(organization))
}

func (h *Handler) Create(c *appctx.APIContext) {
	var req structs.CreateOrgOption
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Validation(c.Context, err, &req)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		appctx.FieldError(c.Context, "name", "required", "name is required")
		return
	}
	organization, err := organization_model.CreateOrganization(c.Request.Context(), h.store, c.Doer.AccountID, req.Name)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToOrganization(organization))
}

func (h *Handler) Edit(c *appctx.APIContext) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	var req structs.EditOrgOption
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Validation(c.Context, err, &req)
		return
	}
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		appctx.FieldError(c.Context, "name", "required", "name is required")
		return
	}
	organization, err := organization_model.UpdateOrganization(c.Request.Context(), h.store, c.Doer.AccountID, id, *req.Name)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToOrganization(organization))
}

func organizationID(c *appctx.APIContext) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.APIErrorAuto(organization_model.ErrInvalidOrganization)
		return 0, false
	}
	return id, true
}
