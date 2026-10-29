package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

type verificationDistributorFake struct {
	err      error
	accounts []int64
}

func (f *verificationDistributorFake) Enqueue(
	_ context.Context, task *asynq.Task, _ ...asynq.Option,
) error {
	if f.err != nil {
		return f.err
	}
	if task.Type() == TaskSendPasswordReset {
        // Another test package may have queued a reset intent in the shared
        // disposable database. The dispatcher itself is type-agnostic.
        var payload PayloadSendPasswordReset
        if err := json.Unmarshal(task.Payload(), &payload); err != nil { return err }
        if payload.AccountID <= 0 { return errors.New("invalid reset account") }
        return nil
    }
    if task.Type() != TaskSendVerifyEmail {
        return errors.New("unexpected task type")
    }
    var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	f.accounts = append(f.accounts, payload.AccountID)
	return nil
}

func TestJobOutboxRetriesRedisFailures(t *testing.T) {
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
		Email:        uuid.NewString() + "@example.test",
		PasswordHash: "test-only",
	})
	if err != nil {
		t.Fatal(err)
	}

	queued := func() bool {
		t.Helper()
		var completed bool
		if err := pool.QueryRow(ctx,
			"SELECT queued_at IS NOT NULL FROM job_outbox WHERE (payload->>'account_id')::bigint = $1",
			account.ID,
		).Scan(&completed); err != nil {
			t.Fatal(err)
		}
		return completed
	}

	fake := &verificationDistributorFake{err: errors.New("Redis unavailable")}
	dispatcher := NewOutboxDispatcher(store, fake)
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
	// If Redis reports an existing stable TaskID (for example after a
	// crash between enqueue and PostgreSQL commit), treat it as enqueued.
	second, err := store.RegisterTx(ctx, db.RegisterTxParams{
		Email:        uuid.NewString() + "@example.test",
		PasswordHash: "test-only",
	})
	if err != nil {
		t.Fatal(err)
	}
	fake.err = asynq.ErrTaskIDConflict
	for i := 0; i < 20; i++ {
		if _, err := dispatcher.Dispatch(ctx); err != nil {
			t.Fatalf("task ID conflict should not cause retry loop: %v", err)
		}
		var completed bool
		if err := pool.QueryRow(ctx,
			"SELECT queued_at IS NOT NULL FROM job_outbox WHERE (payload->>'account_id')::bigint = $1",
			second.ID,
		).Scan(&completed); err != nil {
			t.Fatal(err)
		}
		if completed {
			return
		}
	}
	t.Fatal("stable task ID conflict did not mark the outbox row queued")

}

// taskRecorder accepts arbitrary task types. The dispatcher must not depend
// on email-specific payloads or add a new transport method for each job.
type taskRecorder struct {
	types map[string]int
}

func (r *taskRecorder) Enqueue(_ context.Context, task *asynq.Task, _ ...asynq.Option) error {
	if r.types == nil {
		r.types = make(map[string]int)
	}
	r.types[task.Type()]++
	return nil
}

func TestJobOutboxDispatchesIndependentTaskTypes(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing writes without disposable test mode")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var id int64
	taskType := "task:report:test:" + uuid.NewString()
	if err := pool.QueryRow(ctx,
		"INSERT INTO job_outbox(task_type,payload,queue_name,run_at) VALUES($1,$2::jsonb,'default',now()) RETURNING id",
		taskType, `{"report_id":42}`,
	).Scan(&id); err != nil {
		t.Fatal(err)
	}
	recorder := &taskRecorder{}
	dispatcher := NewOutboxDispatcher(db.NewStore(pool), recorder)
	queued := false
	for i := 0; i < 30 && !queued; i++ {
		if _, err := dispatcher.Dispatch(ctx); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, "SELECT queued_at IS NOT NULL FROM job_outbox WHERE id = $1", id).Scan(&queued); err != nil {
			t.Fatal(err)
		}
	}
	if !queued || recorder.types[taskType] != 1 {
		t.Fatalf("non-verification task was not handed off exactly once in test: queued=%t types=%v", queued, recorder.types)
	}
}
