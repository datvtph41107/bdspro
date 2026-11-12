package db

import (
	sqlc "github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Native pgx pool and transactions satisfy SQLC's DBTX contract.
type engine struct {
	pool    *pgxpool.Pool
	queries *Queries
}

func New(conn DBTX) *Queries { return sqlc.New(conn) }
