package db

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresPoolReadiness(t *testing.T) {
	t.Setenv("PGCONNECT_TIMEOUT", "")
	ctx := context.Background()
	for _, source := range []string{"", "postgres://%zz"} {
		if pool, err := openPool(ctx, source, nil); err == nil || pool != nil {
			t.Fatal("invalid configuration produced a pool")
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if pool, err := openPool(canceled, "postgres://localhost/db", nil); !errors.Is(err, context.Canceled) || pool != nil {
		t.Fatal("canceled context did not stop initialization", err)
	}

	// Local TCP endpoint accepts a connection but never completes PostgreSQL
	// startup. This is deterministic without depending on a network outage.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			<-done
		}
	}()
	source := "postgres://fixture:fixture@" + listener.Addr().String() + "/fixture?sslmode=disable&connect_timeout=1"
	start := time.Now()
	if pool, err := openPool(ctx, source, nil); err == nil || pool != nil {
		t.Fatal("unresponsive PostgreSQL reported ready")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("readiness timeout did not bound the stalled handshake")
	}

	realSource := os.Getenv("BDSPRO_ENGINE_TEST_DATABASE_URL")
	if realSource == "" {
		t.Log("disposable PostgreSQL not configured; successful pool test skipped")
		return
	}
	u, err := url.Parse(realSource)
	if err != nil || u.Scheme != "postgresql" && u.Scheme != "postgres" {
		t.Fatal("test database URL must be a PostgreSQL URL")
	}
	for _, tc := range []struct {
		timeout string
		want    time.Duration
	}{
		{"", defaultConnectTimeout},
		{"3", 3 * time.Second},
	} {
		q := u.Query()
		q.Del("connect_timeout")
		if tc.timeout != "" {
			q.Set("connect_timeout", tc.timeout)
		}
		q.Set("pool_max_conns", "2")
		u.RawQuery = q.Encode()
		pool, err := openPool(ctx, u.String(), nil)
		if err != nil {
			t.Fatal("disposable PostgreSQL not ready")
		}
		if pool.Config().ConnConfig.ConnectTimeout != tc.want || pool.Config().MaxConns != 2 {
			pool.Close()
			t.Fatal("connection timeout or explicit pool sizing changed")
		}
		var value int
		err = pool.QueryRow(ctx, "SELECT 1").Scan(&value)
		pool.Close()
		if err != nil || value != 1 {
			t.Fatal("ready pool did not execute a query")
		}
	}
}

// Run in a dedicated test process: the real engine and graceful manager are
// intentionally process-lifetime singletons, not resettable application state.
func TestDefaultEngineLifecycle(t *testing.T) {
	source := os.Getenv("BDSPRO_ENGINE_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("disposable PostgreSQL not configured")
	}
	ctx := context.Background()
	if _, err := GetEngine(ctx); !errors.Is(err, ErrEngineNotInitialized) {
		t.Fatal("uninitialized engine did not fail closed", err)
	}
	if err := InitEngine(nil); err == nil {
		t.Fatal("nil initialization context accepted")
	}
	if err := InitEngine(ctx); err == nil {
		t.Fatal("engine initialized without a graceful manager")
	}

	managerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	graceful.InitManager(managerCtx)
	g := graceful.GetManager()
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	registered := true
	defer func() {
		if registered {
			g.ServerDone()
		}
		g.DoGracefulShutdown()
		select {
		case <-g.Done():
		case <-time.After(10 * time.Second):
			t.Error("graceful manager did not terminate")
		}
	}()

	previous := setting.Database
	defer func() { setting.Database = previous }()
	setting.Database = setting.DatabaseSettings{DBSource: "postgres://%zz", LogSQL: "off"}
	if err := InitEngine(g.ShutdownContext()); err == nil {
		t.Fatal("invalid database config initialized the engine")
	}
	if _, err := GetQueries(ctx); !errors.Is(err, ErrEngineNotInitialized) {
		t.Fatal("failed initialization published queries", err)
	}
	setting.Database.DBSource = source
	canceled, stop := context.WithCancel(ctx)
	stop()
	if err := InitEngine(canceled); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled initialization did not fail", err)
	}
	if _, err := GetEngine(ctx); !errors.Is(err, ErrEngineNotInitialized) {
		t.Fatal("canceled initialization published a pool", err)
	}
	if err := InitEngine(g.ShutdownContext()); err != nil {
		t.Fatal("engine initialization failed", err)
	}
	if err := InitEngine(g.ShutdownContext()); !errors.Is(err, ErrEngineAlreadyInitialized) {
		t.Fatal("duplicate initialization accepted", err)
	}

	handle, err := GetEngine(ctx)
	if err != nil {
		t.Fatal(err)
	}
	pool, ok := handle.(*pgxpool.Pool)
	if !ok {
		t.Fatal("default engine is not a PostgreSQL pool")
	}
	first, err := GetQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GetQueries(ctx)
	if err != nil || first == nil || second != first {
		t.Fatal("default SQLC queries not shared")
	}
	// A temporary table is visible only on the transaction's actual connection.
	// Reading it through a generated query proves the SQLC handle uses that tx,
	// rather than accidentally escaping to the pool.
	var ended context.Context
	err = WithTx(ctx, func(txCtx context.Context) error {
		ended = txCtx
		conn, err := GetEngine(txCtx)
		if err != nil {
			return err
		}
		q, err := GetQueries(txCtx)
		if err != nil {
			return err
		}
		scope := txCtx.Value(transactionContextKey{}).(*transactionScope)
		if conn != scope.tx || q != scope.queries || q == first || conn == handle {
			t.Fatal("transaction handles do not share one scope")
		}
		child := context.WithValue(txCtx, struct{ fixture bool }{true}, true)
		childConn, err := GetEngine(child)
		if err != nil || childConn != conn {
			t.Fatal("derived context lost transaction handle", err)
		}
		childQueries, err := GetQueries(child)
		if err != nil || childQueries != q {
			t.Fatal("derived context lost SQLC handle", err)
		}
		if _, err := conn.Exec(txCtx, "CREATE TEMP TABLE oauth2_credentials (issuer text, subject text, account_id bigint) ON COMMIT DROP"); err != nil {
			return err
		}
		if _, err := conn.Exec(txCtx, "INSERT INTO pg_temp.oauth2_credentials VALUES ($1, $2, $3)", "fixture-issuer", "fixture-subject", int64(42)); err != nil {
			return err
		}
		account, err := q.GetOAuth2AccountID(txCtx, sqlc.GetOAuth2AccountIDParams{Issuer: "fixture-issuer", Subject: "fixture-subject"})
		if err != nil {
			return err
		}
		if account != 42 {
			t.Fatal("generated query did not read transaction-local data")
		}
		return nil
	})
	if err != nil {
		t.Fatal("transaction query acquisition failed", err)
	}
	if _, err := GetEngine(ended); !errors.Is(err, ErrTransactionClosed) {
		t.Fatal("ended scope fell back to pool", err)
	}
	if _, err := GetQueries(ended); !errors.Is(err, ErrTransactionClosed) {
		t.Fatal("ended scope fell back to default queries", err)
	}
	t.Run("transaction contract", func(t *testing.T) { verifyTransactionContract(t, pool) })
	g.DoGracefulShutdown()
	select {
	case <-g.Done():
		t.Fatal("terminated while server activity was registered")
	default:
	}
	var value int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&value); err != nil || value != 1 {
		t.Fatal("pool closed before server activity finished")
	}
	g.ServerDone()
	registered = false
	select {
	case <-g.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("engine termination did not complete")
	}
	if _, err := GetEngine(ctx); !errors.Is(err, ErrEngineClosed) {
		t.Fatal("closed engine still accessible", err)
	}
	if _, err := GetQueries(ctx); !errors.Is(err, ErrEngineClosed) {
		t.Fatal("closed engine still returns queries", err)
	}
	if err := pool.Ping(ctx); err == nil {
		t.Fatal("underlying pgx pool remained open")
	}
	CloseEngine()
	if err := InitEngine(ctx); !errors.Is(err, ErrEngineClosed) {
		t.Fatal("closed engine reinitialized", err)
	}
}
