package auth

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OIDCClaims carries identity claims AFTER an OpenID Connect ID token was
// cryptographically verified against a trusted provider configuration.
// This value type does not verify or grant account access on its own.
type OIDCClaims struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
}

// Key checks key shape only; it does not validate signature, audience or nonce.
// Preserve the verified issuer exactly, including its path when present.
func (identity OIDCClaims) Key() (OAuth2CredentialKey, error) {
	issuer := identity.Issuer
	parsed, err := url.Parse(issuer)
	if err != nil || parsed == nil || parsed.Scheme != "https" || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		parsed.Opaque != "" || parsed.String() != issuer {
		return OAuth2CredentialKey{}, ErrOIDCIssuerInvalid
	}
	if len(identity.Subject) > 255 {
		return OAuth2CredentialKey{}, ErrOIDCSubjectInvalid
	}
	for i := 0; i < len(identity.Subject); i++ {
		if identity.Subject[i] > 127 {
			return OAuth2CredentialKey{}, ErrOIDCSubjectInvalid
		}
	}
	if strings.TrimSpace(identity.Subject) == "" || identity.Subject != strings.TrimSpace(identity.Subject) {
		return OAuth2CredentialKey{}, ErrOIDCSubjectInvalid
	}
	return OAuth2CredentialKey{Issuer: issuer, Subject: identity.Subject}, nil
}

// OAuth2Credential is the persistent Account-to-issuer binding model.
// Generated SQLC values must not escape models/auth.
type OAuth2Credential struct {
	Issuer    string
	Subject   string
	AccountID int64
	LinkedAt  time.Time
}

func oauth2CredentialFromDB(row db.Oauth2Credential) OAuth2Credential {
	return OAuth2Credential{
		Issuer: row.Issuer, Subject: row.Subject,
		AccountID: row.AccountID, LinkedAt: row.LinkedAt,
	}
}

// OAuth2CredentialKey identifies an external subject within its issuer.
// Neither email nor a provider display name is an account-linking key.
type OAuth2CredentialKey struct {
	Issuer  string
	Subject string
}

// ListOAuth2CredentialsForAccount reads all credentials linked to an Account.
func ListOAuth2CredentialsForAccount(ctx context.Context, queries db.Querier, accountID int64) ([]OAuth2Credential, error) {
	rows, err := queries.ListOAuth2CredentialsForAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	credentials := make([]OAuth2Credential, 0, len(rows))
	for _, row := range rows {
		credentials = append(credentials, oauth2CredentialFromDB(row))
	}
	return credentials, nil
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
		return util.WithCause(ErrOAuth2CredentialAlreadyLinked, err)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		pgErr.ConstraintName == "oauth2_credentials_account_issuer_uq" {
		return util.WithCause(ErrOAuth2CredentialAlreadyLinked, err)
	}
	return err
}
