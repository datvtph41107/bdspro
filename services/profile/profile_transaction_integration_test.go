package profile

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pinnedProfileStore struct {
	db.Store
	conn    *pgxpool.Conn
	ready   chan<- struct{}
	release <-chan struct{}
}

func (s pinnedProfileStore) ExecTx(ctx context.Context, run func(*db.Queries) error) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := run(db.New(tx)); err != nil {
		return err
	}
	if s.ready != nil {
		close(s.ready)
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return tx.Commit(ctx)
}

func disposableProfileStore(t *testing.T) (context.Context, *pgxpool.Pool, *db.SQLStore) {
	t.Helper()
	url := os.Getenv("BDS_TEST_DB_URL")
	if url == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing non-disposable PostgreSQL")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing database other than bdspro_test")
	}
	cfg.MaxConns = max(cfg.MaxConns, 6)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, pool, db.NewStore(pool)
}

func newProfileSession(t *testing.T, ctx context.Context, store *db.SQLStore, accountID int64) uuid.UUID {
	t.Helper()
	sessionID := uuid.New()
	_, err := store.CreateSession(ctx, db.CreateSessionParams{
		ID: sessionID, AccountID: accountID,
		RefreshToken: "sha256:" + strings.Repeat("e", 64) + uuid.NewString(),
		UserAgent:    "profile-transaction-acceptance", ClientIp: "203.0.113.65",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	return sessionID
}

func waitProfileSessionBlock(t *testing.T, ctx context.Context, pool *pgxpool.Pool, blocker, waiter uint32) {
	t.Helper()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case <-ticker.C:
			var blocked bool
			err := pool.QueryRow(ctx,
				"SELECT $1::integer=ANY(pg_blocking_pids($2::integer))",
				int32(blocker), int32(waiter),
			).Scan(&blocked)
			if err != nil {
				t.Fatal(err)
			}
			if blocked {
				return
			}
		case <-timeout.C:
			t.Fatal("PostgreSQL did not exhibit the required session-row lock contention")
		}
	}
}

func TestProfileMutationSerializedWithSessionRevocation(t *testing.T) {
	ctx, pool, store := disposableProfileStore(t)
	account, err := account_model.Register(ctx, store, account_model.RegisterParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "profile-tx-acceptance-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	live := newProfileSession(t, ctx, store, account.ID)
	service := NewService(store)

	initial := "Authorized"
	original, err := service.Update(ctx, account.ID, live, ProfilePatch{DisplayName: &initial})
	if err != nil || original.DisplayName != initial {
		t.Fatalf("initial profile write failed: %+v err=%v", original, err)
	}
	utc := "UTC"
	before, err := service.UpdateSettings(ctx, account.ID, live, SettingsPatch{Timezone: &utc})
	if err != nil || before.Timezone != "UTC" {
		t.Fatalf("initial settings write failed: %+v err=%v", before, err)
	}

	revoker, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer revoker.Release()
	writer, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Release()
	revokerPID := revoker.Conn().PgConn().PID()
	writerPID := writer.Conn().PgConn().PID()
	tx, err := revoker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1", live); err != nil {
		t.Fatal(err)
	}
	stale := "Must not commit"
	result := make(chan error, 1)
	go func() {
		_, err := NewService(pinnedProfileStore{Store: store, conn: writer}).Update(
			ctx, account.ID, live, ProfilePatch{DisplayName: &stale},
		)
		result <- err
	}()
	waitProfileSessionBlock(t, ctx, pool, revokerPID, writerPID)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, auth_model.ErrSessionUnavailable) {
			t.Fatalf("revoked session was allowed to update profile: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("profile mutation did not finish after revocation committed")
	}
	profile, err := service.Show(ctx, account.ID)
	if err != nil || profile.DisplayName != initial {
		t.Fatalf("revoked transaction changed profile: %+v err=%v", profile, err)
	}
	locale := "en"
	if _, err := service.UpdateSettings(ctx, account.ID, live, SettingsPatch{Locale: &locale}); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("revoked session changed settings: %v", err)
	}

	// Opposite order: a transaction that already owns the acting-session
	// lock may commit its profile change; revocation must wait for it.
	nextSession := newProfileSession(t, ctx, store, account.ID)
	held, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	ready := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	done := make(chan error, 1)
	go func() {
		newLocale := "en"
		_, err := NewService(pinnedProfileStore{
			Store: store, conn: held, ready: ready, release: release,
		}).UpdateSettings(ctx, account.ID, nextSession, SettingsPatch{Locale: &newLocale})
		done <- err
	}()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("profile transaction did not reach pre-commit hold")
	}
	revokeConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer revokeConn.Release()
	revokeFinished := make(chan error, 1)
	go func() {
		_, err := revokeConn.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1", nextSession)
		revokeFinished <- err
	}()
	// Prove the lock edge using pinned PostgreSQL backends, not a sleep.
	waitProfileSessionBlock(t, ctx, pool, held.Conn().PgConn().PID(), revokeConn.Conn().PgConn().PID())
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("authorized in-flight settings write failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("profile transaction did not commit")
	}
	select {
	case err := <-revokeFinished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("revocation did not finish after profile transaction")
	}
	after, err := service.Show(ctx, account.ID)
	if err != nil || after.Locale != "en" || after.Timezone != "UTC" || after.DisplayName != initial {
		t.Fatalf("profile/settings transaction commit order violated: %+v err=%v", after, err)
	}
}
