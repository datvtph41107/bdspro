package api

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/gin-gonic/gin"
)

func organizationError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, organization.ErrForbidden), errors.Is(err, organization.ErrNotFound):
		ctx.String(http.StatusNotFound, "resource unavailable")
	case errors.Is(err, organization.ErrInvalid):
		ctx.String(http.StatusBadRequest, "invalid request")
	case errors.Is(err, organization.ErrConflict):
		ctx.String(http.StatusConflict, "conflict")
	default:
		ctx.String(http.StatusInternalServerError, "internal server error")
	}
}

func (server *Server) organizationID(ctx *gin.Context, label string) (int64, bool) {
	id, err := auth.PositiveID(ctx.Param("id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "invalid %s", label)
		return 0, false
	}
	return id, true
}

type organizationRegistrationRequest struct {
	Name             string `json:"name"`
	Jurisdiction     string `json:"jurisdiction"`
	RegistrationCode string `json:"registration_code"`
}

func (server *Server) submitOrganizationRegistration(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	var req organizationRegistrationRequest
	if !bindJSON(ctx, &req, 16<<10) {
		ctx.String(http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := server.organizations.SubmitRegistration(ctx.Request.Context(), actor, req.Name, req.Jurisdiction, req.RegistrationCode)
	if err != nil {
		organizationError(ctx, err)
		return
	}
	replyJSON(ctx, http.StatusCreated, gin.H{"request_id": id, "status": "PENDING"})
}

type organizationClaimRequest struct {
	EvidenceReference string `json:"evidence_reference"`
}

func (server *Server) submitOrganizationClaim(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	orgID, ok := server.organizationID(ctx, "organization id")
	if !ok {
		return
	}
	var req organizationClaimRequest
	if !bindJSON(ctx, &req, 16<<10) {
		ctx.String(http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := server.organizations.SubmitClaim(ctx.Request.Context(), actor, orgID, req.EvidenceReference)
	if err != nil {
		organizationError(ctx, err)
		return
	}
	replyJSON(ctx, http.StatusCreated, gin.H{"request_id": id, "status": "PENDING"})
}

type organizationApprovalRequest struct {
	ReviewReference string `json:"review_reference"`
}

func (server *Server) approveOrganizationRegistration(ctx *gin.Context) {
	reviewer, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	id, ok := server.organizationID(ctx, "request id")
	if !ok {
		return
	}
	var req organizationApprovalRequest
	if !bindJSON(ctx, &req, 16<<10) {
		ctx.String(http.StatusBadRequest, "invalid request body")
		return
	}
	orgID, err := server.organizations.ApproveRegistration(ctx.Request.Context(), reviewer, id, req.ReviewReference)
	if err != nil {
		organizationError(ctx, err)
		return
	}
	replyJSON(ctx, http.StatusOK, gin.H{"organization_id": orgID, "status": "APPROVED"})
}

func (server *Server) approveOrganizationClaim(ctx *gin.Context) {
	reviewer, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	id, ok := server.organizationID(ctx, "request id")
	if !ok {
		return
	}
	var req organizationApprovalRequest
	if !bindJSON(ctx, &req, 16<<10) {
		ctx.String(http.StatusBadRequest, "invalid request body")
		return
	}
	orgID, err := server.organizations.ApproveInitialClaim(ctx.Request.Context(), reviewer, id, req.ReviewReference)
	if err != nil {
		organizationError(ctx, err)
		return
	}
	replyJSON(ctx, http.StatusOK, gin.H{"organization_id": orgID, "status": "APPROVED"})
}

type organizationInvitationRequest struct {
	Email  string `json:"email"`
	RoleID int64  `json:"role_id"`
}

func (server *Server) inviteOrganizationMember(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	orgID, ok := server.organizationID(ctx, "org id")
	if !ok {
		return
	}
	var req organizationInvitationRequest
	if !bindJSON(ctx, &req, 16<<10) {
		ctx.String(http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := server.organizations.Invite(ctx.Request.Context(), actor, orgID, req.RoleID, req.Email)
	if err != nil {
		organizationError(ctx, err)
		return
	}
	replyJSON(ctx, http.StatusCreated, gin.H{"invitation_id": id, "status": "PENDING"})
}

func (server *Server) acceptOrganizationInvitation(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	id, ok := server.organizationID(ctx, "invitation id")
	if !ok {
		return
	}
	membershipID, err := server.organizations.AcceptInvitation(ctx.Request.Context(), actor, id)
	if err != nil {
		organizationError(ctx, err)
		return
	}
	replyJSON(ctx, http.StatusOK, gin.H{"membership_id": membershipID, "status": "ACTIVE"})
}

func (server *Server) listOrganizationMembers(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	orgID, ok := server.organizationID(ctx, "org id")
	if !ok {
		return
	}
	members, err := server.organizations.ListMembers(ctx.Request.Context(), actor, orgID)
	if err != nil {
		organizationError(ctx, err)
		return
	}
	replyJSON(ctx, http.StatusOK, members)
}
