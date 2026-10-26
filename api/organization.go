package api

import (
    apirequest "github.com/datvtph41107/bdspro/app/http/requests/api"
	"errors"
	"net/http"
	"strings"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	orgapp "github.com/datvtph41107/bdspro/organization"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type createOrganizationRequest = apirequest.CreateOrganizationBody

func (server *Server) createOrganization(ctx *gin.Context) {
	var req createOrganizationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		invalidJSON(ctx, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	actorAccountID := authenticatedAccountID(ctx)
	organization, err := orgapp.CreateForAccount(ctx.Request.Context(), server.store, actorAccountID, name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "create organization failed"})
		return
	}
	ctx.JSON(http.StatusCreated, organization)
}

func (server *Server) listOrganizations(ctx *gin.Context) {
	actorAccountID := authenticatedAccountID(ctx)
	organizations, err := orgapp.ListForAccount(ctx.Request.Context(), server.store, actorAccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "list organizations failed"})
		return
	}
	ctx.JSON(http.StatusOK, organizations)
}

type addOrganizationMemberRequest = apirequest.AddOrganizationMemberBody

func (server *Server) addOrganizationMember(ctx *gin.Context) {
	organizationID, validID := positiveInt64Param(ctx, "id")
	if !validID {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization"})
		return
	}
	var req addOrganizationMemberRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		invalidJSON(ctx, err)
		return
	}
	actorAccountID := authenticatedAccountID(ctx)
	member, err := server.store.AddOrganizationMember(ctx.Request.Context(), db.AddOrganizationMemberParams{
		OrganizationID: organizationID,
		PersonID:       req.PersonID,
		ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "organization owner required"})
		return
	}
	if err != nil {
		if pgErr := postgresError(err); pgErr != nil {
			if pgErr.Code == "23505" {
				ctx.JSON(http.StatusConflict, gin.H{"error": "membership already exists"})
				return
			}
			if pgErr.Code == "23503" {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "person unavailable"})
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "add member failed"})
		return
	}
	ctx.JSON(http.StatusCreated, member)
}

type updateOrganizationRequest = apirequest.UpdateOrganizationBody

func (server *Server) updateOrganization(ctx *gin.Context) {
	organizationID, validID := positiveInt64Param(ctx, "id")
	if !validID {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization"})
		return
	}
	var req updateOrganizationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		invalidJSON(ctx, err)
		return
	}
	if req.Name == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	name := strings.TrimSpace(*req.Name)
	if name == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	actorAccountID := authenticatedAccountID(ctx)
	organization, err := server.store.UpdateOrganization(ctx.Request.Context(), db.UpdateOrganizationParams{
		OrganizationID: organizationID,
		ActorAccountID: actorAccountID,
		Name:           pgtype.Text{String: name, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "organization owner required"})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "update organization failed"})
		return
	}
	ctx.JSON(http.StatusOK, organization)
}
