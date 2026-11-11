package oauth

import (
	oauth_model "github.com/datvtph41107/bdspro/models/oauth"
	"context"

	"github.com/google/uuid"
)

// AuthorizedClient is a currently active account-scoped authorization.
// Scope remains fixed to profile:read in the restricted current server.
type AuthorizedClient struct {
	ClientID uuid.UUID
	Name     string
	Scope    string
}

func (s *Service) ListAuthorizations(ctx context.Context, accountID int64) ([]AuthorizedClient, error) {
	if accountID <= 0 {
		return nil, ErrAccessDenied
	}
	clients, err := oauth_model.ListOAuthAuthorizations(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]AuthorizedClient, 0, len(clients))
	for _, c := range clients {
		out = append(out, AuthorizedClient{ClientID: c.ClientID, Name: c.Name, Scope: ScopeProfile})
	}
	return out, nil
}

func (s *Service) RevokeAuthorization(ctx context.Context, accountID int64, clientID uuid.UUID) error {
	if accountID <= 0 || clientID == uuid.Nil {
		return ErrInvalidRequest
	}
	return oauth_model.RevokeOAuthAuthorization(ctx, accountID, clientID)
}
