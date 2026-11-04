package db

import (
	"context"

	sqlc "github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store exposes SQLC queries and the existing transaction operations.
type Store interface {
	Querier
	PhoneOTPRepository
	GetAccountByID(ctx context.Context, id int64) (Account, error)
	GetAccountByEmail(ctx context.Context, email string) (Account, error)
	ExecTx(ctx context.Context, fn func(*Queries) error) error
	StartPhoneSignup(ctx context.Context, phone string, mac []byte, devCode string) error
	CompletePhoneSignup(ctx context.Context, phone string, mac []byte) (Account, error)
	LinkGoogleIdentity(ctx context.Context, arg LinkGoogleIdentityParams) error
	GetGoogleAccountID(ctx context.Context, googleSub string) (int64, error)
}

type SQLStore struct {
	pool *pgxpool.Pool
	*Queries
}

// New exposes SQLC query construction through the stable db package.
func New(conn DBTX) *Queries {
	return sqlc.New(conn)
}

func NewStore(pool *pgxpool.Pool) *SQLStore {
	return &SQLStore{pool: pool, Queries: New(pool)}
}

func (s *SQLStore) Close() {
	s.pool.Close()
}

// ExecTx owns transaction creation, commit and rollback.
func (s *SQLStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err = fn(New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ Store = (*SQLStore)(nil)
