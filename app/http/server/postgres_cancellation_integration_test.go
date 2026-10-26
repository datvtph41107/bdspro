package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const arc02SlowQuery = "SELECT 1 FROM pg_sleep(8) /* bdspro_arc02_cancellation */"

// arc02SlowStore substitutes only the first Organization Store query. It uses
// real PostgreSQL I/O, but never inserts or updates business data.
type arc02SlowStore struct {
	db.Store
	pool    *pgxpool.Pool
	started chan uint32
	done    chan error
}

func (s *arc02SlowStore) GetAccountByID(ctx context.Context, _ int64) (db.Account, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		s.done <- err
		return db.Account{}, err
	}
	defer conn.Release()

	s.started <- conn.Conn().PgConn().PID()
	var result int
	err = conn.QueryRow(ctx, arc02SlowQuery).Scan(&result)
	s.done <- err
	if err != nil {
		return db.Account{}, err
	}
	// A successful slow query should never continue to a real business write.
	return db.Account{}, errors.New("ARC-02 probe completed without cancellation")
}

func TestARC02HTTPDisconnectCancelsPostgresQuery(t *testing.T) {
	// This test runs only against the disposable PostgreSQL service provided
	// by GitHub Actions. In particular, it must not connect to the WSL dev DB.
	dsn := os.Getenv("ARC02_CANCEL_TEST_DB_URL")
	if os.Getenv("GITHUB_ACTIONS") != "true" || dsn == "" {
		t.Skip("real HTTP/PG cancellation requires the disposable GitHub Actions PostgreSQL service")
	}
	target, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", target.Hostname())
	require.Equal(t, "5432", target.Port())
	require.Equal(t, "/bdspro", target.Path)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	poolConfig.MaxConns = 3 // one active probe plus separate observer connections
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	var database string
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT current_database()").Scan(&database))
	require.Equal(t, "bdspro", database)

	spy := &arc02SlowStore{
		pool:    pool,
		started: make(chan uint32, 1),
		done:    make(chan error, 1),
	}
	app := newTestServer(t, spy)
	accessToken, _, err := app.tokenMaker.CreateToken(19, time.Minute, token.TokenTypeAccess)
	require.NoError(t, err)

	// httptest.NewServer opens a real TCP listener and uses the actual
	// CORS -> Gin -> authenticated Organization handler pipeline.
	httpServer := httptest.NewServer(app.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, httpServer.URL+"/organizations",
		strings.NewReader(`{"name":"arc02-cancellation"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 12 * time.Second}
	clientDone := make(chan error, 1)
	go func() {
		resp, err := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		clientDone <- err
	}()

	var pid uint32
	select {
	case pid = <-spy.started:
	case <-time.After(3 * time.Second):
		t.Fatal("Organization handler never reached the PostgreSQL query")
	}

	// Confirm that PostgreSQL is actually executing pg_sleep, rather than
	// cancelling before the driver has sent the query.
	observeCtx, stopObserve := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopObserve()
	require.Eventually(t, func() bool {
		var active bool
		err := pool.QueryRow(observeCtx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE pid = $1 AND state = 'active'
				  AND query LIKE '%bdspro_arc02_cancellation%'
			)`, int32(pid)).Scan(&active)
		return err == nil && active
	}, 2500*time.Millisecond, 40*time.Millisecond, "slow query was not observed as active in PostgreSQL")

	startCancel := time.Now()
	cancel() // cancel real HTTP client request while the PG query is active

	select {
	case err := <-spy.done:
		require.Error(t, err, "PostgreSQL query unexpectedly completed without cancellation")
		require.Less(t, time.Since(startCancel), 3*time.Second,
			"request cancellation should interrupt an 8-second PostgreSQL query")
	case <-time.After(3 * time.Second):
		t.Fatal("PostgreSQL I/O did not return promptly after HTTP request cancellation")
	}

	select {
	case err := <-clientDone:
		require.Error(t, err, "the cancelled HTTP client should not receive a successful response")
	case <-time.After(3 * time.Second):
		t.Fatal("the HTTP client was not released after cancellation")
	}

	// The cancelled pg_sleep should no longer be active at the observed PID;
	// pgx may close the connection instead of reusing it after cancellation.
	checkCtx, stopCheck := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCheck()
	require.Eventually(t, func() bool {
		var active bool
		err := pool.QueryRow(checkCtx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE pid = $1 AND state = 'active'
				  AND query LIKE '%bdspro_arc02_cancellation%'
			)`, int32(pid)).Scan(&active)
		return err == nil && !active
	}, 2500*time.Millisecond, 40*time.Millisecond, "cancelled PostgreSQL query is still active")
}
