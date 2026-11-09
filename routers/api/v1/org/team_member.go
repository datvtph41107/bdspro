package org

import (
	"net/http"

	org_model "github.com/datvtph41107/bdspro/models/organization"
	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	org_service "github.com/datvtph41107/bdspro/services/organization"
)

// ListTeamMembers reads account-scoped membership through the domain model.
// No duplicate teamAssignment: SQLC enforces access and empty-page visibility.
func ListTeamMembers(ctx *context.APIContext) {
	teamID, err := ctx.PositivePathID("team_id")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	options, ok := teamListOptions(ctx)
	if !ok {
		return
	}
	members, err := org_model.ListTeamUsersForDoer(ctx, teamID, options)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToTeamMembers(members))
}

// AddTeamMember parses HTTP input; the service locks the live acting session
// and SQLC enforces owner/recipient membership in the same transaction.
func AddTeamMember(ctx *context.APIContext) {
	teamID, err := ctx.PositivePathID("team_id")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	form := web.GetForm[*api.AddTeamUserOption](ctx)
	member, err := org_service.AddTeamMember(ctx, teamID, form.PersonID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToTeamMember(member))
}

// RemoveTeamMember owns HTTP only; audit and deletion commit together in SQLC.
func RemoveTeamMember(ctx *context.APIContext) {
	teamID, err := ctx.PositivePathID("team_id")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	personID, err := ctx.PositivePathID("person_id")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	if err := org_service.RemoveTeamMember(ctx, teamID, personID); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
