// Package pgerrors provides narrow SQLSTATE/constraint classification.
// A SQLSTATE alone does not identify the business meaning of a failure.
package pgerrors

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	UniqueViolation = "23505"
	ForeignKeyViolation = "23503"
	CheckViolation = "23514"
	SerializationFailure = "40001"
	DeadlockDetected = "40P01"
	LockNotAvailable = "55P03"
)

// HasCode inspects PgError through wrapped and joined errors; callers must
// still decide whether the error implies a business contract or is transient.
func HasCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

// UniqueConstraint matches only a named database invariant known to belong
// to this operation. It deliberately does not translate every 23505 to 409.
func UniqueConstraint(err error, names ...string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != UniqueViolation {
		return false
	}
	for _, name := range names {
		if name != "" && pgErr.ConstraintName == name {
			return true
		}
	}
	return false
}

// RetryableTransactionFailure recognizes errors that can justify restarting
// the ENTIRE transaction if and only if the caller's operation is idempotent.
// This is classification only; never retry after an ambiguous COMMIT outcome.
func RetryableTransactionFailure(err error) bool {
	return HasCode(err, SerializationFailure) || HasCode(err, DeadlockDetected)
}
