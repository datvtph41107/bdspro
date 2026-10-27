package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	apirequest "github.com/datvtph41107/bdspro/api/handler/request/organization"
	"github.com/datvtph41107/bdspro/api/service"
	apperrors "github.com/datvtph41107/bdspro/core/errors"
	middleware "github.com/datvtph41107/bdspro/core/middleware"
	"github.com/datvtph41107/bdspro/core/response"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// OrganizationHandler owns HTTP actions, not SQL/transactions.
type OrganizationHandler struct{ organizations *service.OrganizationService }

func NewOrganizationHandler(organizations *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{organizations: organizations}
}

func (c *OrganizationHandler) Create(ctx *gin.Context) {
	var input apirequest.CreateOrganizationBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		response.Send(ctx, http.StatusBadRequest, nil, err, "invalid request")
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		response.Send(ctx, 0, nil, apperrors.NameRequired, "")
		return
	}
	organization, err := c.organizations.Create(ctx.Request.Context(), middleware.AccountID(ctx), name)
	response.Send(ctx, http.StatusCreated, organization, err, "create organization failed")
}

func (c *OrganizationHandler) Index(ctx *gin.Context) {
	organizations, err := c.organizations.List(ctx.Request.Context(), middleware.AccountID(ctx))
	response.Send(ctx, http.StatusOK, organizations, err, "list organizations failed")
}

func (c *OrganizationHandler) AddMember(ctx *gin.Context) {
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
	member, err := c.organizations.AddMember(ctx.Request.Context(), middleware.AccountID(ctx), id, req.PersonID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = apperrors.OrganizationOwnerRequired
	} else if pgErr := apperrors.Postgres(err); pgErr != nil {
		if pgErr.Code == "23505" {
			err = apperrors.Wrap(apperrors.MembershipExists, err)
		}
		if pgErr.Code == "23503" {
			err = apperrors.Wrap(apperrors.PersonUnavailable, err)
		}
	}
	response.Send(ctx, http.StatusCreated, member, err, "add member failed")
}

func (c *OrganizationHandler) Update(ctx *gin.Context) {
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
	name := strings.TrimSpace(*req.Name)
	if name == "" {
		response.Send(ctx, 0, nil, apperrors.NameRequired, "")
		return
	}
	organization, err := c.organizations.Update(ctx.Request.Context(), middleware.AccountID(ctx), id, name)
	if errors.Is(err, pgx.ErrNoRows) {
		err = apperrors.OrganizationOwnerRequired
	}
	response.Send(ctx, http.StatusOK, organization, err, "update organization failed")
}
