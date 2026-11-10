package auth

import (
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

// OAuthClient is the authorization-server application model. It intentionally
// excludes SecretHash, revocation columns and SQLC's pgtype fields. Unlike
// the external OIDC provider client, this represents a locally registered
// OAuth application, comparable to Gitea's models/auth.OAuth2Application.
type OAuthClient struct {
	ID             uuid.UUID
	OwnerAccountID int64
	Name           string
	RedirectURI    string
	CreatedAt      time.Time
}

// OAuthClientFromDB is the only authorized SQLC-to-domain projection.
// This function must never copy credential hashes to an HTTP-facing result.
func OAuthClientFromDB(row db.OauthClient) *OAuthClient {
	return &OAuthClient{
		ID:             row.ID,
		OwnerAccountID: row.OwnerAccountID,
		Name:           row.Name,
		RedirectURI:    row.RedirectUri,
		CreatedAt:      row.CreatedAt,
	}
}
