package db

import (
	"context"

	sqlc "github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// New exposes SQLC query construction through the stable db package.
func New(conn DBTX) *Queries {
	return sqlc.New(conn)
}

// contextDBTX is a DBTX executor selector, not a second transaction owner.
// It is installed only on an ingress SQLStore. In a transaction callback,
// the same pgx.Tx opened by SQLStore.ExecTxRaw is selected from txCtx;
// ordinary requests continue to use the pool. SQLC-generated code stays
// unchanged and both paths retain the same DBTX method signatures.
type contextDBTX struct {
	pool *pgxpool.Pool
}

func (d contextDBTX) executor(ctx context.Context) (DBTX, error) {
	if ctx.Value(queriesContextKey{}) == nil {
		return d.pool, nil
	}
	if _, err := TxQueries(ctx); err != nil {
		return nil, err
	}
	// A context from SQLStore A must not redirect SQLStore B to A's pool.
	// Bind by pool identity so two stores for the same pool can interoperate,
	// while a different database/tenant fails closed.
	if ctx.Value(rawTxStoreContextKey{}) == nil || ctx.Value(rawTxStoreContextKey{}) != d.pool {
		if state, ok := ctx.Value(txStateContextKey{}).(*transactionState); ok {
			state.fail(ErrTxStoreUnavailable)
		}
		return nil, ErrTxStoreUnavailable
	}
	tx, ok := ctx.Value(rawTxContextKey{}).(pgx.Tx)
	if !ok || tx == nil {
		if state, ok := ctx.Value(txStateContextKey{}).(*transactionState); ok {
			state.fail(ErrTxStoreUnavailable)
		}
		return nil, ErrTxStoreUnavailable
	}
	return tx, nil
}

func (d contextDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	exec, err := d.executor(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	return exec.Exec(ctx, sql, args...)
}

func (d contextDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	exec, err := d.executor(ctx)
	if err != nil {
		return nil, err
	}
	return exec.Query(ctx, sql, args...)
}

type queryErrorRow struct{ err error }

func (r queryErrorRow) Scan(...any) error { return r.err }

func (d contextDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	exec, err := d.executor(ctx)
	if err != nil {
		return queryErrorRow{err: err}
	}
	return exec.QueryRow(ctx, sql, args...)
}

var _ DBTX = contextDBTX{}

// transactionDBTX wraps the owning pgx.Tx for SQLC statements that receive
// a *Queries directly from TxQueries. It forbids a caller from retaining the
// executor but silently supplying the original (non-transactional) context.
// The state pointer is the authority; the pgx.Tx itself is never exposed to
// SQLC callers as an unguarded DBTX.
type transactionDBTX struct {
	tx    pgx.Tx
	state *transactionState
	pool  *pgxpool.Pool
}

func (d transactionDBTX) executor(ctx context.Context) (DBTX, error) {
	if ctx.Value(txStateContextKey{}) != d.state ||
		ctx.Value(queriesContextKey{}) == nil ||
		ctx.Value(rawTxStoreContextKey{}) != d.pool {
		d.state.fail(ErrTransactionRequired)
		return nil, ErrTransactionRequired
	}
	if err := d.state.failure(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		d.state.fail(err)
		return nil, err
	}
	return d.tx, nil
}

func (d transactionDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	exec, err := d.executor(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	return exec.Exec(ctx, sql, args...)
}

func (d transactionDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	exec, err := d.executor(ctx)
	if err != nil {
		return nil, err
	}
	return exec.Query(ctx, sql, args...)
}

func (d transactionDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	exec, err := d.executor(ctx)
	if err != nil {
		return queryErrorRow{err: err}
	}
	return exec.QueryRow(ctx, sql, args...)
}

var _ DBTX = transactionDBTX{}

