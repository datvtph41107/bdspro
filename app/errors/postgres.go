package apperrors

import (
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres returns driver details for operation-specific classification.
// It deliberately does not assign HTTP meaning to PostgreSQL codes.
func Postgres(err error) *pgconn.PgError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr
	}
	return nil
}
