package auth

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// phoneRegisteredConflict recognizes only the globally unique verified phone.
func phoneRegisteredConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "phone_credentials_phone_e164_key"
}

// phoneCredentialConflict recognizes the account and phone uniqueness
// constraints. Other unique violations must not be treated as phone conflicts.
func phoneCredentialConflict(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return pgErr.ConstraintName == "phone_credentials_pkey" ||
		pgErr.ConstraintName == "phone_credentials_phone_e164_key"
}
