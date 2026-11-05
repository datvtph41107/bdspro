package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A real PG18 test, refusing to touch anything but a disposable test database.
func TestAccountConnectionHistoryAtomicSessionAndIsolation(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("BDS_TEST_MODE=disposable is mandatory")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing writes outside disposable test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	createAccount := func() Account {
		t.Helper()
		account, err := WithTxResult(ctx, store, func(q *Queries) (Account, error) {
			personID, err := CreatePersonWithParty(ctx, q)
			if err != nil {
				return Account{}, err
			}
			account, err := q.CreateAccount(ctx, CreateAccountParams{
				PersonID: personID, Email: "history-" + uuid.NewString() + "@example.test",
			})
			if err != nil {
				return Account{}, err
			}
			return account, q.CreateAccountSettings(ctx, account.ID)
		})
		if err != nil {
			t.Fatal(err)
		}
		return account
	}
	first, second := createAccount(), createAccount()
	params := CreateSessionParams{
		ID: uuid.New(), AccountID: first.ID,
		RefreshToken: "sha256:" + strings.Repeat("a", 64),
		UserAgent: "DRX account-history integration", ClientIp: "203.0.113.9",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	session, err := store.CreateSessionWithConnection(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	history, err := store.ListAccountConnectionHistories(ctx, first.ID)
	if err != nil || len(history) != 1 {
		t.Fatalf("login audit missing: %+v, %v", history, err)
	}
	if history[0].SessionID != session.ID || history[0].Action != "login" ||
		history[0].IpAddress != params.ClientIp || history[0].UserAgent != params.UserAgent {
		t.Fatalf("wrong history: %+v", history[0])
	}
	others, err := store.ListAccountConnectionHistories(ctx, second.ID)
	if err != nil || len(others) != 0 {
		t.Fatalf("cross-account leak: %+v, %v", others, err)
	}
	// The sessions table accepts text, but the history table rejects IP
	// strings longer than the longest IPv6 presentation. History failure
	// must roll back the otherwise valid refresh-session insert.
	bad := params
	bad.ID = uuid.New()
	bad.ClientIp = strings.Repeat("x", 46)
	if _, err := store.CreateSessionWithConnection(ctx, bad); err == nil {
		t.Fatal("invalid connection history unexpectedly committed")
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE id=$1", bad.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("orphan session survived failed login audit transaction")
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM account_connection_histories WHERE account_id=$1", first.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("failed audit added a row: %d", count)
	}
	if err := store.DeleteSessionsForAccount(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	history, err = store.ListAccountConnectionHistories(ctx, first.ID)
	if err != nil || len(history) != 1 || history[0].SessionID != session.ID {
		t.Fatalf("history did not survive session cleanup: %+v, %v", history, err)
	}
}
