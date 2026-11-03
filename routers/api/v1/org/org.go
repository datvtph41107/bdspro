package org

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
	apictx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
)

// API is an explicit dependency owner; it does not use mutable package-global
// stores or implicit ORM sessions. Its methods follow Gitea's org endpoint naming.
type API struct{ store organization_model.Store }

func NewAPI(store organization_model.Store) *API { return &API{store: store} }

func actorID(ctx *apictx.APIContext) (int64, bool) {
	if ctx.Doer == nil {
		ctx.AuthenticationRequired()
		return 0, false
	}
	return ctx.Doer.AccountID, true
}

func organizationID(ctx *apictx.APIContext) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(ctx.Req, "id"), 10, 64)
	if err != nil || id <= 0 {
		ctx.APIError(http.StatusBadRequest, "http.invalid_organization", "invalid organization")
		return 0, false
	}
	return id, true
}

func (a *API) List(ctx *apictx.APIContext) {
	actor, ok := actorID(ctx)
	if !ok {
		return
	}
	result, err := organization_model.ListForAccount(ctx, a.store, actor)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganizationAccessList(result))
}

func (a *API) Get(ctx *apictx.APIContext) {
	actor, ok := actorID(ctx)
	if !ok {
		return
	}
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	access, err := organization_model.GetForAccount(ctx, a.store, actor, id)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganizationAccess(access))
}

func (a *API) Create(ctx *apictx.APIContext) {
	actor, ok := actorID(ctx)
	if !ok {
		return
	}
	var form structs.CreateOrgOption
	if !ctx.BindJSON(&form) {
		return
	}
	if strings.TrimSpace(form.Name) == "" {
		ctx.FieldError("name", "required", "name is required")
		return
	}
	created, err := organization_model.CreateOrganization(ctx, a.store, actor, form.Name)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToOrganization(created))
}

func (a *API) Edit(ctx *apictx.APIContext) {
	actor, ok := actorID(ctx)
	if !ok {
		return
	}
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	var form structs.EditOrgOption
	if !ctx.BindJSON(&form) {
		return
	}
	if form.Name == nil || strings.TrimSpace(*form.Name) == "" {
		ctx.FieldError("name", "required", "name is required")
		return
	}
	updated, err := organization_model.UpdateOrganization(ctx, a.store, actor, id, *form.Name)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganization(updated))
}

func (a *API) Members(ctx *apictx.APIContext) {
	actor, ok := actorID(ctx)
	if !ok {
		return
	}
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	members, err := organization_model.GetOrgMembers(ctx, a.store, actor, id)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToActiveOrganizationMembers(members))
}

func (a *API) AddMember(ctx *apictx.APIContext) {
	actor, ok := actorID(ctx)
	if !ok {
		return
	}
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	var form structs.AddOrgUserOption
	if !ctx.BindJSON(&form) {
		return
	}
	if form.PersonID <= 0 {
		ctx.FieldError("person_id", "required", "person_id is required")
		return
	}
	member, err := organization_model.AddOrgUser(ctx, a.store, actor, id, form.PersonID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToOrganizationMember(member))
}

var _ organization_model.Store = (db.Store)(nil)
