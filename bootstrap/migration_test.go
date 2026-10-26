package bootstrap

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Verify startup migration handling only on disposable CI/local integration databases.
func TestRunDBMigrationNoChange(t *testing.T) {
	inCI := os.Getenv("GITHUB_ACTIONS") == "true"
	if !inCI && os.Getenv("BDSPRO_INTEGRATION_DB") != "1" {
		t.Skip("migration integration test requires CI or make test-integration")
	}

	config, err := util.LoadConfig("..")
	require.NoError(t, err)
	if !inCI {
		target, err := url.Parse(config.DBSource)
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1", target.Hostname())
		require.Equal(t, "15433", target.Port())
		require.Equal(t, "/bdspro", target.Path)

		pool, err := pgxpool.New(context.Background(), config.DBSource)
		require.NoError(t, err)
		t.Cleanup(pool.Close)
		var cluster string
		require.NoError(t, pool.QueryRow(context.Background(), "SHOW cluster_name").Scan(&cluster))
		require.Equal(t, "bdspro-disposable", cluster, "refusing to migrate non-disposable PostgreSQL")
	}

	require.NoError(t, RunDBMigration("file://../db/migration", config.DBSource))
}
