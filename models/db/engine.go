package db

import (
	"context"

	sqlc "github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type engine struct {
	pool    *pgxpool.Pool
	queries *Queries
}

// New preserves the generated SQLC constructor in the models/db package.
func New(conn DBTX) *Queries { return sqlc.New(conn) }

// poolDBTX is used only outside transactions. Using it with a transaction
// context must fail instead of silently escaping that transaction.
type poolDBTX struct{ pool *pgxpool.Pool }

func (d poolDBTX) executor(ctx context.Context) (DBTX, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if value := ctx.Value(transactionContextKey{}); value != nil {
		if scope, ok := value.(*transactionScope); ok && scope != nil {
			scope.fail(ErrTransactionContext)
		}
		return nil, ErrTransactionContext
	}
	return d.pool, nil
}

func (d poolDBTX) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	executor, err := d.executor(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	return executor.Exec(ctx, query, args...)
}

func (d poolDBTX) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	executor, err := d.executor(ctx)
	if err != nil {
		return nil, err
	}
	return executor.Query(ctx, query, args...)
}

type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }

func (d poolDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	executor, err := d.executor(ctx)
	if err != nil {
		return errorRow{err: err}
	}
	return executor.QueryRow(ctx, query, args...)
}

// transactionDBTX belongs to precisely one transaction scope. SQLC query
// execution must not accept a parent or unrelated context.
type transactionDBTX struct{ scope *transactionScope }

func (d transactionDBTX) executor(ctx context.Context) (pgx.Tx, error) {
	if ctx.Value(transactionContextKey{}) != d.scope {
		d.scope.fail(ErrTransactionContext)
		if other, ok := ctx.Value(transactionContextKey{}).(*transactionScope); ok && other != nil {
			other.fail(ErrTransactionContext)
		}
		return nil, ErrTransactionContext
	}
	if err := d.scope.failure(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		d.scope.fail(err)
		return nil, err
	}
	return d.scope.tx, nil
}

func (d transactionDBTX) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tx, err := d.executor(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	tag, err := tx.Exec(ctx, query, args...)
	d.scope.fail(err)
	return tag, err
}

func (d transactionDBTX) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	tx, err := d.executor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, query, args...)
	d.scope.fail(err)
	return rows, err
}

func (d transactionDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	tx, err := d.executor(ctx)
	if err != nil {
		return errorRow{err: err}
	}
	return tx.QueryRow(ctx, query, args...)
}

var (
	_ DBTX    = poolDBTX{}
	_ DBTX    = transactionDBTX{}
	_ pgx.Row = errorRow{}
)
