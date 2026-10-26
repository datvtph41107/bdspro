package api

import (
    apperrors "github.com/datvtph41107/bdspro/app/errors"
    "github.com/jackc/pgx/v5/pgconn"
)

func postgresError(err error) *pgconn.PgError {
    return apperrors.Postgres(err)
}
