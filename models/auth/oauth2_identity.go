package auth

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

// GetOAuth2AccountID reads a verified external identity binding.
// A missing identity returns the query's error unchanged.
func GetOAuth2AccountID(ctx context.Context, queries db.Querier, key IdentityKey) (int64, error) {
	if _, err := (OIDCIdentity{Issuer: key.Issuer, Subject: key.Subject}).Key(); err != nil {
		return 0, err
	}
	return queries.GetOAuth2AccountID(ctx, db.GetOAuth2AccountIDParams{
		Issuer: key.Issuer, Subject: key.Subject,
	})
}

// LinkOAuth2Identity binds an identity to an existing account only after a
// caller has authenticated the user and verified the provider identity.
// PostgreSQL enforces that an existing identity cannot move to another account.
func LinkOAuth2Identity(ctx context.Context, queries db.Querier, accountID int64, key IdentityKey) error {
	if _, err := (OIDCIdentity{Issuer: key.Issuer, Subject: key.Subject}).Key(); err != nil {
		return err
	}
	_, err := queries.LinkOAuth2Identity(ctx, db.LinkOAuth2IdentityParams{
		Issuer: key.Issuer, Subject: key.Subject, AccountID: accountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOAuth2IdentityAlreadyLinked
	}
	return err
}
