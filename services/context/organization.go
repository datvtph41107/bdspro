package context

import (
	"errors"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Actor represents a JWT access token backed by an active database session.
// A signed token alone is insufficient to authenticate a request.
type Actor struct {
	AccountID int64
	SessionID uuid.UUID
}

// OrganizationResource is the verified organization membership for a request.
// Authorization for state changes is repeated inside SQL transactions.
type OrganizationResource struct {
	ID   int64
	Name string
	Role string
}

// AuthenticateAccessToken runs after APIContexter, before any organization
// loader or handler. Do not trust account IDs supplied in HTTP headers.
func AuthenticateAccessToken(ctx *APIContext) {
	if ctx.Actor != nil {
		return
	}
	header := ctx.CurrentRequest().Header.Get("Authorization")
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		ctx.AuthenticationRequired()
		return
	}
	payload, err := auth.ProcessMaker().VerifyToken(parts[1], auth.TokenTypeAccess)
	if err != nil || payload.SessionID == uuid.Nil {
		ctx.AuthenticationRequired()
		return
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	session, err := q.GetSession(ctx, payload.SessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.AuthenticationRequired()
		return
	}
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	if session.AccountID != payload.AccountID || session.IsBlocked || !session.ExpiresAt.After(time.Now()) {
		ctx.AuthenticationRequired()
		return
	}
	ctx.Actor = &Actor{AccountID: payload.AccountID, SessionID: session.ID}
}

// AssignOrganization resolves an organization only for an authenticated
// member. It intentionally returns 404 when membership is absent.
func AssignOrganization(ctx *APIContext) {
	if ctx.Actor == nil {
		ctx.AuthenticationRequired()
		return
	}
	id, err := ctx.PositivePathID("org")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	org, err := q.GetOrganizationForAccount(ctx, db.GetOrganizationForAccountParams{
		AccountID: ctx.Actor.AccountID, OrganizationID: id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.APIErrorNotFound()
		return
	}
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.Org = &OrganizationResource{ID: org.PartyID, Name: org.Name, Role: org.Role}
}
