package db

import (
    "context"
    "os"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *pgxpool.Pool {
    t.Helper()
    dsn := os.Getenv("DB_TEST_URL")
    if dsn == "" { t.Skip("DB_TEST_URL is unset: PostgreSQL integration test not run") }
    pool, err := pgxpool.New(context.Background(), dsn)
    require.NoError(t, err)
    t.Cleanup(pool.Close)
    var name string
    require.NoError(t, pool.QueryRow(context.Background(), "SELECT current_database()").Scan(&name))
    require.Equal(t, "bdspro_lecture37_test", name, "refusing to write another database")
    return pool
}
