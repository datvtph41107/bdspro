package handler

import (
	"net/http"
	"strconv"

	apirequest "github.com/datvtph41107/bdspro/api/handler/request/organization"
	"github.com/datvtph41107/bdspro/api/service"
	apperrors "github.com/datvtph41107/bdspro/core/errors"
	"github.com/datvtph41107/bdspro/core/middleware"
	"github.com/datvtph41107/bdspro/core/response"
	"github.com/gin-gonic/gin"
)

// OrganizationHandler binds HTTP input and projects service outcomes.
type OrganizationHandler struct{ organizations *service.OrganizationService }

func NewOrganizationHandler(organizations *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{organizations: organizations}
}

func (h *OrganizationHandler) Create(ctx *gin.Context) {
	var input apirequest.CreateOrganizationBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	organization, err := h.organizations.Create(ctx.Request.Context(), middleware.AccountID(ctx), input.Name)
	response.Send(ctx, http.StatusCreated, organization, err, "create organization failed")
}

func (h *OrganizationHandler) Index(ctx *gin.Context) {
	organizations, err := h.organizations.List(ctx.Request.Context(), middleware.AccountID(ctx))
	response.Send(ctx, http.StatusOK, organizations, err, "list organizations failed")
}

func (h *OrganizationHandler) AddMember(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Send(ctx, 0, nil, apperrors.InvalidOrganization, "")
		return
	}
	var req apirequest.AddOrganizationMemberBody
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	member, err := h.organizations.AddMember(ctx.Request.Context(), middleware.AccountID(ctx), id, req.PersonID)
	response.Send(ctx, http.StatusCreated, member, err, "add member failed")
}

func (h *OrganizationHandler) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Send(ctx, 0, nil, apperrors.InvalidOrganization, "")
		return
	}
	var req apirequest.UpdateOrganizationBody
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	if req.Name == nil {
		response.Send(ctx, 0, nil, apperrors.NameRequired, "")
		return
	}
	organization, err := h.organizations.Update(ctx.Request.Context(), middleware.AccountID(ctx), id, *req.Name)
	response.Send(ctx, http.StatusOK, organization, err, "update organization failed")
}
