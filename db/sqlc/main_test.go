package db

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// openTestDB permits writes only on the disposable GitHub Actions DB or the
// dedicated local integration-test PostgreSQL (:15433, disposable cluster).
// Both instances use the canonical database name "bdspro".
func openTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	inCI := os.Getenv("GITHUB_ACTIONS") == "true"
	inLocalIntegration := os.Getenv("BDSPRO_INTEGRATION_DB") == "1"
	if !inCI && !inLocalIntegration {
		t.Skip("PostgreSQL write tests require CI or make test-integration")
	}

	config, err := util.LoadConfig("../..")
	require.NoError(t, err)

	if !inCI {
		target, err := url.Parse(config.DBSource)
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1", target.Hostname(), "refusing to test against a remote database")
		require.Equal(t, "15433", target.Port(), "refusing to test against the development database")
		require.Equal(t, "/bdspro", target.Path)
	}

	pool, err := pgxpool.New(context.Background(), config.DBSource)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	var name string
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT current_database()").Scan(&name))
	require.Equal(t, "bdspro", name)

	if !inCI {
		var cluster string
		require.NoError(t, pool.QueryRow(context.Background(), "SHOW cluster_name").Scan(&cluster))
		require.Equal(t, "bdspro-disposable", cluster, "refusing to write into a non-disposable PostgreSQL cluster")
	}
	return pool
}
