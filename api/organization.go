package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type createOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

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
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.GetAccountByID(ctx.Request.Context(), payload.AccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "create organization failed"})
		return
	}
	organization, err := server.store.CreateOrganizationTx(ctx.Request.Context(), db.CreateOrganizationTxParams{
		Name: name, OwnerPersonID: account.PersonID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "create organization failed"})
		return
	}
	ctx.JSON(http.StatusCreated, organization)
}

func (server *Server) listOrganizations(ctx *gin.Context) {
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.GetAccountByID(ctx.Request.Context(), payload.AccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "list organizations failed"})
		return
	}
	organizations, err := server.store.ListOrganizationsForPerson(ctx.Request.Context(), account.PersonID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "list organizations failed"})
		return
	}
	ctx.JSON(http.StatusOK, organizations)
}

type addOrganizationMemberRequest struct {
	PersonID int64 `json:"person_id" binding:"required,min=1"`
}

func (server *Server) addOrganizationMember(ctx *gin.Context) {
	organizationID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || organizationID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization"})
		return
	}
	var req addOrganizationMemberRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		invalidJSON(ctx, err)
		return
	}
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	member, err := server.store.AddOrganizationMember(ctx.Request.Context(), db.AddOrganizationMemberParams{
		OrganizationID: organizationID,
		PersonID:       req.PersonID,
		ActorAccountID: payload.AccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "organization owner required"})
		return
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
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

type updateOrganizationRequest struct {
	Name *string `json:"name"`
}

func (server *Server) updateOrganization(ctx *gin.Context) {
	organizationID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || organizationID <= 0 {
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
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	organization, err := server.store.UpdateOrganization(ctx.Request.Context(), db.UpdateOrganizationParams{
		OrganizationID: organizationID,
		ActorAccountID: payload.AccountID,
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
