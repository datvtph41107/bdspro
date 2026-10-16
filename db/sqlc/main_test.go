package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var testDB *pgxpool.Pool

// TestMain follows SimpleBank's one database-connection setup. The environment
// variable is intentionally opt-in: an ordinary `go test ./...` cannot write
// to the developer database accidentally.
func TestMain(m *testing.M) {
	dsn := os.Getenv("SQLC_TEST_DATABASE_URL")
	if dsn == "" {
		os.Exit(m.Run()) // Each SQLC test skips when no database is configured.
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid SQLC_TEST_DATABASE_URL:", err)
		os.Exit(1)
	}

	host := config.ConnConfig.Host
	allowedDatabase := config.ConnConfig.Database == "bdspro" || config.ConnConfig.Database == "bdspro_sqlc_ci"
	if !allowedDatabase ||
		(host != "localhost" && host != "127.0.0.1" && host != "::1") {
		fmt.Fprintln(os.Stderr, "SQLC tests require local bdspro or disposable bdspro_sqlc_ci")
		os.Exit(1)
	}

	ctx := context.Background()
	testDB, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot open SQLC test database:", err)
		os.Exit(1)
	}
	if err := testDB.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "cannot reach SQLC test database:", err)
		testDB.Close()
		os.Exit(1)
	}

	var databaseName string
	if err := testDB.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName); err != nil || (databaseName != "bdspro" && databaseName != "bdspro_sqlc_ci") {
		fmt.Fprintln(os.Stderr, "unexpected SQLC test database:", databaseName, err)
		testDB.Close()
		os.Exit(1)
	}

	code := m.Run()
	testDB.Close()
	os.Exit(code)
}

// Each test gets an independent transaction. Never call Commit here:
// the tests are allowed to run against the user's local bdspro database.
func beginTestTx(t *testing.T) (context.Context, pgx.Tx, *Queries) {
	t.Helper()
	if testDB == nil {
		t.Skip("set SQLC_TEST_DATABASE_URL to run PostgreSQL SQLC tests")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return ctx, tx, New(tx)
}
