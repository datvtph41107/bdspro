package worker

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

type verificationDistributorFake struct {
	err error
	accounts []int64
}

func (f *verificationDistributorFake) DistributeTaskSendVerifyEmail(
	_ context.Context, payload *PayloadSendVerifyEmail, _ ...asynq.Option,
) error {
	if f.err != nil {
		return f.err
	}
	f.accounts = append(f.accounts, payload.AccountID)
	return nil
}

func TestVerificationOutboxRetriesRedisFailures(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable bdspro_test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes without BDS_TEST_MODE=disposable")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing database writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	store := db.NewStore(pool)
	account, err := store.RegisterTx(ctx, db.RegisterTxParams{
		Email: uuid.NewString()+"@example.test",
		PasswordHash: "test-only",
	})
	if err != nil {
		t.Fatal(err)
	}

	queued := func() bool {
		t.Helper()
		var completed bool
		if err := pool.QueryRow(ctx,
			"SELECT queued_at IS NOT NULL FROM verification_outbox WHERE account_id = $1",
			account.ID,
		).Scan(&completed); err != nil {
			t.Fatal(err)
		}
		return completed
	}

	fake := &verificationDistributorFake{err: errors.New("Redis unavailable")}
	dispatcher := NewVerificationDispatcher(store, fake)
	if _, err := dispatcher.Dispatch(ctx); err == nil {
		t.Fatal("Redis failure must be returned so outbox stays pending")
	}
	if queued() {
		t.Fatal("Redis failure incorrectly marked intent as queued")
	}
	fake.err = nil

	// A previous package may leave additional pending outbox entries in the
	// same disposable CI database. Drain bounded batches until ours is processed.
	for i := 0; i < 20 && !queued(); i++ {
		if _, err := dispatcher.Dispatch(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if !queued() {
		t.Fatal("outbox row was never queued after Redis recovered")
	}
	sent := 0
	for _, id := range fake.accounts {
		if id == account.ID {
			sent++
		}
	}
	if sent != 1 {
		t.Fatalf("expected one enqueue for account %d, got %d", account.ID, sent)
	}
}
