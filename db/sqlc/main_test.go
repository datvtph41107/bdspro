package db

import (
	"context"
	"os"
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// openTestDB runs write-capable integration tests only on an isolated Actions runner.
// Both environments call the database bdspro; the local database is never a test target.
func openTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("database integration tests are limited to GitHub Actions to protect local bdspro")
	}

	config, err := util.LoadConfig("../..")
	require.NoError(t, err)

	pool, err := pgxpool.New(context.Background(), config.DBSource)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	var name string
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT current_database()").Scan(&name))
	require.Equal(t, "bdspro", name, "refusing to run integration tests against another database")

	return pool
}
