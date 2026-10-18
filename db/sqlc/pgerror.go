package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresError unwraps a PostgreSQL server error, including one returned
// through an ExecTx errors.Join or a service's fmt.Errorf("%w", err).
// A connection failure, context cancellation, or pgx.ErrNoRows is not a
// PostgreSQL server SQLSTATE and returns nil.
func PostgresError(err error) *pgconn.PgError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr
	}
	return nil
}
