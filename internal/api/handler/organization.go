package handler

import (
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/datvtph41107/bdspro/internal/api/response"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/datvtph41107/bdspro/internal/service"
	"github.com/gin-gonic/gin"
)

type OrganizationHandler struct {
	organizations *service.OrganizationService
}

func NewOrganizationHandler(organizations *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{organizations: organizations}
}

func (h *OrganizationHandler) Create(c *gin.Context) {
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	organization, err := h.organizations.Create(c.Request.Context(), middleware.AccountID(c), req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, newOrganizationResponse(organization))
}

func (h *OrganizationHandler) Index(c *gin.Context) {
	organizations, err := h.organizations.List(c.Request.Context(), middleware.AccountID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, newOrganizationListResponse(organizations))
}

func (h *OrganizationHandler) AddMember(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, apperrors.InvalidOrganization)
		return
	}
	var req AddOrganizationMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	member, err := h.organizations.AddMember(c.Request.Context(), middleware.AccountID(c), id, req.PersonID)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, newOrganizationMemberResponse(member))
}

func (h *OrganizationHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, apperrors.InvalidOrganization)
		return
	}
	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if req.Name == nil {
		response.Error(c, apperrors.NameRequired)
		return
	}
	organization, err := h.organizations.Update(c.Request.Context(), middleware.AccountID(c), id, *req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, newOrganizationResponse(organization))
}
