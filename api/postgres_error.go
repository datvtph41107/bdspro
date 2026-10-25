package api

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// postgresError unwraps a PostgreSQL error without deciding its HTTP meaning.
// Callers own the code/constraint checks and the public error response.
func postgresError(err error) *pgconn.PgError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr
	}
	return nil
}
