package auth

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OAuth2Credential is the persistent Account-to-issuer binding. The alias
// exposes the expected OAuth2 capitalization without copying the SQLC model.
type OAuth2Credential = db.Oauth2Credential

// OAuth2CredentialKey identifies an external subject within its issuer.
// Neither email nor a provider display name is an account-linking key.
type OAuth2CredentialKey struct {
	Issuer  string
	Subject string
}

// ListOAuth2CredentialsForAccount reads all credentials linked to an Account.
func ListOAuth2CredentialsForAccount(ctx context.Context, queries db.Querier, accountID int64) ([]OAuth2Credential, error) {
	return queries.ListOAuth2CredentialsForAccount(ctx, accountID)
}

// GetOAuth2AccountID reads a verified external identity binding.
// A missing identity returns the query's error unchanged.
func GetOAuth2AccountID(ctx context.Context, queries db.Querier, key OAuth2CredentialKey) (int64, error) {
	if _, err := (OIDCClaims{Issuer: key.Issuer, Subject: key.Subject}).Key(); err != nil {
		return 0, err
	}
	return queries.GetOAuth2AccountID(ctx, db.GetOAuth2AccountIDParams{
		Issuer: key.Issuer, Subject: key.Subject,
	})
}

// LinkOAuth2Credential binds an identity to an existing account only after a
// caller has authenticated the user and verified the provider identity.
// PostgreSQL enforces that an existing identity cannot move to another account.
func LinkOAuth2Credential(ctx context.Context, queries db.Querier, accountID int64, key OAuth2CredentialKey) error {
	if _, err := (OIDCClaims{Issuer: key.Issuer, Subject: key.Subject}).Key(); err != nil {
		return err
	}
	_, err := queries.LinkOAuth2Credential(ctx, db.LinkOAuth2CredentialParams{
		Issuer: key.Issuer, Subject: key.Subject, AccountID: accountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOAuth2CredentialAlreadyLinked
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		pgErr.ConstraintName == "oauth2_credentials_account_issuer_uq" {
		return ErrOAuth2CredentialAlreadyLinked
	}
	return err
}
