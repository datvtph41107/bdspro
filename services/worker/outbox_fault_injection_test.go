package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/queue"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The isolated schema copies the migrated table definition rather than
// inventing a test-specific replacement for the production outbox schema.
// Dedicated search_path ensures these concurrency tests cannot enqueue
// pending rows created by other packages sharing bdspro_test.
func isolatedOutboxPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires migrated disposable PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing schema mutation without disposable test mode")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing schema mutation outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if err := admin.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	schema := "outbox_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	qualified := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+qualified); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+qualified+" CASCADE"); err != nil {
			t.Errorf("drop isolated outbox schema: %v", err)
		}
	})
	if _, err := admin.Exec(ctx, "CREATE TABLE "+qualified+".job_outbox (LIKE public.job_outbox INCLUDING ALL)"); err != nil {
		t.Fatal(err)
	}
	isolatedCfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if isolatedCfg.ConnConfig.RuntimeParams == nil {
		isolatedCfg.ConnConfig.RuntimeParams = make(map[string]string)
	}
	isolatedCfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, isolatedCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return pool
}

func addOutboxTestIntent(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (int64, string) {
	t.Helper()
	taskType := "task:test:outbox:" + uuid.NewString()
	var id int64
	err := pool.QueryRow(ctx, `
        INSERT INTO job_outbox (task_type, payload, queue_name, max_retry, run_at)
        VALUES ($1, '{}'::jsonb, 'default', 10, now()) RETURNING id`, taskType).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id, taskType
}

func outboxQueued(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id int64) bool {
	t.Helper()
	var queued bool
	if err := pool.QueryRow(ctx, "SELECT queued_at IS NOT NULL FROM job_outbox WHERE id = $1", id).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	return queued
}

var errOutboxCommitInjected = errors.New("injected postgres rollback after accepted enqueue")

type rollbackAfterEnqueueStore struct {
	db.Store
	rollbackOnce bool
}

func (s *rollbackAfterEnqueueStore) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	return s.Store.ExecTx(ctx, func(q *db.Queries) error {
		if err := fn(q); err != nil {
			return err
		}
		if s.rollbackOnce {
			s.rollbackOnce = false
			return errOutboxCommitInjected
		}
		return nil
	})
}

// The fake acts like an Asynq producer that retained the task while the
// PostgreSQL transaction rolled back. The replay reports TaskID conflict.
type replayConflictEnqueuer struct {
	taskType string
	attempts int
	accepted int
}

func (e *replayConflictEnqueuer) Enqueue(_ context.Context, task *asynq.Task, _ ...asynq.Option) error {
	if task.Type() != e.taskType {
		return fmt.Errorf("unexpected task type %s", task.Type())
	}
	e.attempts++
	if e.accepted > 0 {
		return asynq.ErrTaskIDConflict
	}
	e.accepted++
	return nil
}

func TestOutboxAcceptedEnqueueBeforeDatabaseRollbackIsRecovered(t *testing.T) {
	pool := isolatedOutboxPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	id, taskType := addOutboxTestIntent(t, ctx, pool)
	store := &rollbackAfterEnqueueStore{Store: db.NewStore(pool), rollbackOnce: true}
	producer := &replayConflictEnqueuer{taskType: taskType}
	dispatcher := NewOutboxDispatcher(store, producer)

	count, err := dispatcher.Dispatch(ctx)
	if count != 0 || !errors.Is(err, errOutboxCommitInjected) {
		t.Fatalf("rollback falsely published queued count: count=%d err=%v", count, err)
	}
	if outboxQueued(t, ctx, pool, id) {
		t.Fatal("uncommitted queue marker survived rollback")
	}
	if producer.attempts != 1 || producer.accepted != 1 {
		t.Fatalf("first delivery not accepted: %+v", producer)
	}

	// The second pass sees the same pending row. A stable Asynq TaskID
	// conflicts, which still lets PostgreSQL commit the queued marker.
	count, err = NewOutboxDispatcher(db.NewStore(pool), producer).Dispatch(ctx)
	if err != nil || count != 1 || !outboxQueued(t, ctx, pool, id) {
		t.Fatalf("replay failed to commit durable marker: count=%d err=%v", count, err)
	}
	if producer.attempts != 2 || producer.accepted != 1 {
		t.Fatalf("replay should not create a second accepted task: %+v", producer)
	}
	count, err = NewOutboxDispatcher(db.NewStore(pool), producer).Dispatch(ctx)
	if err != nil || count != 0 || producer.attempts != 2 {
		t.Fatalf("already queued row dispatched again: count=%d err=%v", count, err)
	}
}

type blockingOutboxEnqueuer struct {
	entered chan struct{}
	release chan struct{}
}

func (e *blockingOutboxEnqueuer) Enqueue(ctx context.Context, _ *asynq.Task, _ ...asynq.Option) error {
	close(e.entered)
	select {
	case <-e.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type unexpectedOutboxEnqueuer struct{ calls int }

func (e *unexpectedOutboxEnqueuer) Enqueue(_ context.Context, _ *asynq.Task, _ ...asynq.Option) error {
	e.calls++
	return nil
}

func TestOutboxConcurrentDispatchSkipsClaimedRow(t *testing.T) {
	pool := isolatedOutboxPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	id, _ := addOutboxTestIntent(t, ctx, pool)
	firstProducer := &blockingOutboxEnqueuer{entered: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(firstProducer.release) }) }
	defer release()
	firstDone := make(chan error, 1)
	go func() {
		_, err := NewOutboxDispatcher(db.NewStore(pool), firstProducer).Dispatch(ctx)
		firstDone <- err
	}()

	select {
	case <-firstProducer.entered:
	case <-ctx.Done():
		t.Fatalf("first dispatcher never acquired row and reached enqueue: %v", ctx.Err())
	}
	secondProducer := &unexpectedOutboxEnqueuer{}
	secondCtx, secondCancel := context.WithTimeout(ctx, 2*time.Second)
	defer secondCancel()
	count, err := NewOutboxDispatcher(db.NewStore(pool), secondProducer).Dispatch(secondCtx)
	if err != nil || count != 0 || secondProducer.calls != 0 {
		t.Fatalf("locked row was not skipped: count=%d calls=%d err=%v", count, secondProducer.calls, err)
	}
	release()
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatalf("first dispatcher could not commit after release: %v", ctx.Err())
	}
	if !outboxQueued(t, ctx, pool, id) {
		t.Fatal("first dispatcher did not durably mark its row after release")
	}
}


// TestOutboxRealRedisOutageThenRecovery exercises the production Asynq
// producer rather than a fake. No durable outbox intent may be acknowledged
// while Redis is unreachable; the same intent must publish after recovery.
// Redis DB 15 and a unique queue/task ID avoid touching normal queues.
func TestOutboxRealRedisOutageThenRecovery(t *testing.T) {
	redisAddr := os.Getenv("BDS_TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("requires disposable Redis and migrated PostgreSQL")
	}
	pool := isolatedOutboxPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	taskID := "g0-outbox-" + uuid.NewString()
	queueName := "g0_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	var id int64
	err := pool.QueryRow(ctx, `
        INSERT INTO job_outbox (task_type, payload, queue_name, max_retry, run_at, task_id)
        VALUES ('task:test:redis-recovery', '{}'::jsonb, $1, 0, now(), $2)
        RETURNING id`, queueName, taskID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	store := db.NewStore(pool)
	unavailable := queue.NewRedisEnqueuer(asynq.RedisClientOpt{Addr: "127.0.0.1:1", DB: 15})
	defer unavailable.Close()
	count, err := NewOutboxDispatcher(store, unavailable).Dispatch(ctx)
	if err == nil || count != 0 {
		t.Fatalf("Redis outage was acknowledged: count=%d err=%v", count, err)
	}
	if outboxQueued(t, ctx, pool, id) {
		t.Fatal("Redis outage committed queued_at")
	}
	recovered := queue.NewRedisEnqueuer(asynq.RedisClientOpt{Addr: redisAddr, DB: 15})
	defer recovered.Close()
	count, err = NewOutboxDispatcher(store, recovered).Dispatch(ctx)
	if err != nil || count != 1 || !outboxQueued(t, ctx, pool, id) {
		t.Fatalf("outbox did not recover: count=%d err=%v", count, err)
	}
	count, err = NewOutboxDispatcher(store, recovered).Dispatch(ctx)
	if err != nil || count != 0 {
		t.Fatalf("committed outbox intent was dispatched again: count=%d err=%v", count, err)
	}
}
