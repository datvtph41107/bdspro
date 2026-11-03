package org

import (
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/services/convert"
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/datvtph41107/bdspro/modules/errors"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	store organization_model.Store
}

func NewHandler(store organization_model.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) List(c *gin.Context) {
	organizations, err := organization_model.ListForAccount(c.Request.Context(), h.store, middleware.AccountID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToOrganizationAccessList(organizations))
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	organization, err := organization_model.GetForAccount(c.Request.Context(), h.store, middleware.AccountID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToOrganizationAccess(organization))
}

func (h *Handler) Create(c *gin.Context) {
	var req structs.CreateOrgOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.FieldError(c, "name", "required", "name is required")
		return
	}
	organization, err := organization_model.CreateOrganization(c.Request.Context(), h.store, middleware.AccountID(c), req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToOrganization(organization))
}

func (h *Handler) Edit(c *gin.Context) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	var req structs.EditOrgOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		response.FieldError(c, "name", "required", "name is required")
		return
	}
	organization, err := organization_model.UpdateOrganization(c.Request.Context(), h.store, middleware.AccountID(c), id, *req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToOrganization(organization))
}

func (h *Handler) Members(c *gin.Context) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	members, err := organization_model.GetOrgMembers(c.Request.Context(), h.store, middleware.AccountID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToActiveOrganizationMembers(members))
}

func (h *Handler) AddMember(c *gin.Context) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	var req structs.AddOrgUserOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	member, err := organization_model.AddOrgUser(c.Request.Context(), h.store, middleware.AccountID(c), id, req.PersonID)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToOrganizationMember(member))
}

func organizationID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, apperrors.InvalidOrganization)
		return 0, false
	}
	return id, true
}
