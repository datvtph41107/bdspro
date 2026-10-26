package api

import (
    "errors"
    "net/http"
    "strings"

    apperrors "github.com/datvtph41107/bdspro/app/errors"
    "github.com/datvtph41107/bdspro/app/http/binding"
    middleware "github.com/datvtph41107/bdspro/app/http/middleware"
    apirequest "github.com/datvtph41107/bdspro/app/http/requests/api"
    "github.com/datvtph41107/bdspro/app/http/responses"
    "github.com/datvtph41107/bdspro/app/services"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgtype"
)

// OrganizationController owns HTTP actions, not SQL/transactions.
type OrganizationController struct { store db.Store }

func NewOrganizationController(store db.Store) *OrganizationController {
    return &OrganizationController{store:store}
}

func (c *OrganizationController) Create(ctx *gin.Context) {
    input := binding.Input[apirequest.CreateOrganizationBody](ctx, "organization.create")
    name := strings.TrimSpace(input.Name)
    if name == "" {
        responses.Error(ctx, apperrors.NameRequired, "")
        return
    }
    organization, err := services.CreateForAccount(ctx.Request.Context(), c.store, middleware.AccountID(ctx), name)
    responses.Send(ctx, http.StatusCreated, organization, err, "create organization failed")
}

func (c *OrganizationController) Index(ctx *gin.Context) {
    organizations, err := services.ListForAccount(ctx.Request.Context(), c.store, middleware.AccountID(ctx))
    responses.Send(ctx, http.StatusOK, organizations, err, "list organizations failed")
}

func (c *OrganizationController) AddMember(ctx *gin.Context) {
    id, ok := binding.PositiveInt64Param(ctx, "id")
    if !ok {
        responses.Error(ctx, apperrors.InvalidOrganization, "")
        return
    }
    req := binding.Input[apirequest.AddOrganizationMemberBody](ctx, "organization.member")
    member, err := c.store.AddOrganizationMember(ctx.Request.Context(), db.AddOrganizationMemberParams{
        OrganizationID:id, PersonID:req.PersonID, ActorAccountID:middleware.AccountID(ctx),
    })
    if errors.Is(err, pgx.ErrNoRows) {
        err = apperrors.OrganizationOwnerRequired
    } else if pgErr := apperrors.Postgres(err); pgErr != nil {
        if pgErr.Code == "23505" { err = apperrors.Wrap(apperrors.MembershipExists, err) }
        if pgErr.Code == "23503" { err = apperrors.Wrap(apperrors.PersonUnavailable, err) }
    }
    responses.Send(ctx, http.StatusCreated, member, err, "add member failed")
}

func (c *OrganizationController) Update(ctx *gin.Context) {
    id, ok := binding.PositiveInt64Param(ctx, "id")
    if !ok {
        responses.Error(ctx, apperrors.InvalidOrganization, "")
        return
    }
    req := binding.Input[apirequest.UpdateOrganizationBody](ctx, "organization.update")
    if req.Name == nil {
        responses.Error(ctx, apperrors.NameRequired, "")
        return
    }
    name := strings.TrimSpace(*req.Name)
    if name == "" {
        responses.Error(ctx, apperrors.NameRequired, "")
        return
    }
    organization, err := c.store.UpdateOrganization(ctx.Request.Context(), db.UpdateOrganizationParams{
        OrganizationID:id, ActorAccountID:middleware.AccountID(ctx),
        Name:pgtype.Text{String:name, Valid:true},
    })
    if errors.Is(err, pgx.ErrNoRows) { err = apperrors.OrganizationOwnerRequired }
    responses.Send(ctx, http.StatusOK, organization, err, "update organization failed")
}
