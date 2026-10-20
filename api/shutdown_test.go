package api

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type httpOutcome struct {
	status int
	body   string
	err    error
}

func runRequest(client *http.Client, url string, out chan<- httpOutcome) {
	resp, err := client.Get(url)
	if err != nil {
		out <- httpOutcome{err: err}
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	out <- httpOutcome{status: resp.StatusCode, body: string(body), err: err}
}

func awaitShutdown(t *testing.T, server *Server) {
	t.Helper()
	require.Eventually(t, server.draining.Load, 2*time.Second, time.Millisecond)
}

func TestGracefulShutdownWaitsForAcceptedRequest(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	entered := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	server.router.GET("/in-flight", func(ctx *gin.Context) {
		close(entered)
		<-release
		ctx.String(http.StatusOK, "completed")
	})

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.serve(runCtx, listener, time.Second) }()
	client := &http.Client{Timeout: 3 * time.Second}
	out := make(chan httpOutcome, 1)
	go runRequest(client, "http://"+listener.Addr().String()+"/in-flight", out)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("request never entered handler")
	}

	cancel()
	awaitShutdown(t, server)
	select {
	case err := <-done:
		t.Fatalf("shutdown returned before the in-flight request completed: %v", err)
	default:
	}
	// The handler was already admitted before the shutdown signal.
	close(release)
	select {
	case result := <-out:
		require.NoError(t, result.err)
		require.Equal(t, http.StatusOK, result.status)
		require.Equal(t, "completed", result.body)
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight request did not complete")
	}
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("graceful shutdown did not finish")
	}
}

func TestShutdownRejectsNewWorkButHealthRemainsLive(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	server.draining.Store(true)
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/health", http.StatusOK},
		{"/ready", http.StatusServiceUnavailable},
		{"/auth/me", http.StatusServiceUnavailable},
		{"/organizations/7/members", http.StatusServiceUnavailable},
		{"/not-registered", http.StatusServiceUnavailable},
	} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		server.Handler().ServeHTTP(recorder, req)
		require.Equal(t, tc.status, recorder.Code, tc.path)
		require.NotEmpty(t, recorder.Header().Get("X-Request-ID"))
	}
}

func TestShutdownDeadlineClosesActiveConnection(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	entered := make(chan struct{})
	abandoned := make(chan struct{})
	server.router.GET("/slow-work", func(ctx *gin.Context) {
		close(entered)
		<-ctx.Request.Context().Done()
		close(abandoned)
	})
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.serve(runCtx, listener, 50*time.Millisecond) }()
	out := make(chan httpOutcome, 1)
	go runRequest(&http.Client{Timeout: 3 * time.Second}, "http://"+listener.Addr().String()+"/slow-work", out)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("slow request never entered")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown deadline was not enforced")
	}
	select {
	case <-abandoned:
	case <-time.After(3 * time.Second):
		t.Fatal("forced close did not cancel the request context")
	}
	select {
	case <-out:
	case <-time.After(3 * time.Second):
		t.Fatal("client was not released after forced close")
	}
}

type brokenListener struct{}

func (brokenListener) Accept() (net.Conn, error) { return nil, errors.New("forced listener failure") }
func (brokenListener) Close() error              { return nil }
func (brokenListener) Addr() net.Addr            { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)} }

func TestUnexpectedHTTPServeFailureIsNotHidden(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	err := server.serve(context.Background(), brokenListener{}, time.Second)
	require.ErrorContains(t, err, "forced listener failure")
	require.False(t, server.draining.Load())
}

func TestShutdownDrainsPostgresTransactionBeforePoolClose(t *testing.T) {
	dsn := os.Getenv("CORE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires CORE_TEST_DATABASE_URL")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	defer pool.Close()
	var name string
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT current_database()").Scan(&name))
	if name != "bdspro_core_ci" {
		t.Fatalf("refusing to use database %q", name)
	}

	server := NewServer(pool, nil, nil, nil, true, "")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	entered := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	server.router.GET("/tx-in-flight", func(ctx *gin.Context) {
		tx, err := pool.Begin(ctx.Request.Context())
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(context.Background())
		if _, err := tx.Exec(ctx.Request.Context(), "SELECT 1"); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}
		close(entered)
		<-release
		if err := tx.Commit(ctx.Request.Context()); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}
		ctx.String(http.StatusOK, "committed")
	})

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.serve(runCtx, listener, time.Second) }()
	out := make(chan httpOutcome, 1)
	go runRequest(&http.Client{Timeout: 3 * time.Second},
		"http://"+listener.Addr().String()+"/tx-in-flight", out)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("transaction never started")
	}
	cancel()
	awaitShutdown(t, server)
	require.EqualValues(t, 1, pool.Stat().AcquiredConns(),
		"connection must remain leased until the request finishes")
	close(release)
	select {
	case result := <-out:
		require.NoError(t, result.err)
		require.Equal(t, http.StatusOK, result.status)
		require.True(t, strings.Contains(result.body, "committed"))
	case <-time.After(3 * time.Second):
		t.Fatal("transaction response did not complete")
	}
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
	require.EqualValues(t, 0, pool.Stat().AcquiredConns(),
		"all transactions must release their connections before main closes pool")
}
