package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrEngineAlreadyInitialized = errors.New("db: engine already initialized")
	ErrEngineNotInitialized     = errors.New("db: engine not initialized")
	ErrEngineClosed             = errors.New("db: engine closed")
	ErrTransactionContext       = errors.New("db: invalid transaction context")
	ErrTransactionClosed        = errors.New("db: transaction closed")
	ErrTransactionCallback      = errors.New("db: nil transaction callback")
	errTransactionPanicked      = errors.New("db: nested transaction callback panicked")
)

// IsUniqueViolation reports a violation of the named PostgreSQL unique constraint or index.
// The caller owns the business meaning of that constraint.
func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return constraint != "" && errors.As(err, &pgErr) && pgErr != nil &&
		pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
