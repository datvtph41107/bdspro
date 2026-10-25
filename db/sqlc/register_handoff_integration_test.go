package db

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

// This is a real PG/Redis integration test of RegisterTx's AfterCreate
// boundary, not a test of a PostgreSQL COMMIT failure or a network partition.
// A deliberate callback error after a successful Redis enqueue forces a PG
// rollback and demonstrates that Redis is outside that transaction.
func TestRegisterTxRedisTaskPersistsAcrossPostgresRollback(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("REDIS_TEST_ADDRESS") == "" {
		t.Skip("requires GitHub Actions disposable PostgreSQL and Redis")
	}

	pool := openTestDB(t)
	redisOpt := asynq.RedisClientOpt{Addr: os.Getenv("REDIS_TEST_ADDRESS")}
	client := asynq.NewClient(redisOpt)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	inspector := asynq.NewInspector(redisOpt)
	t.Cleanup(func() { require.NoError(t, inspector.Close()) })

	for _, tc := range []struct {
		name     string
		abort    bool
		expected int
	}{
		{name: "CommitAfterEnqueue", expected: 1},
		{name: "RollbackAfterEnqueue", abort: true, expected: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			queue := "handoff_" + strings.ReplaceAll(uuid.NewString(), "-", "")
			// Unique test-owned queue; no task worker listens to it.
			t.Cleanup(func() { require.NoError(t, inspector.DeleteQueue(queue, true)) })
			email := uuid.NewString() + "@example.test"
			var inserted Account
			var taskID string
			abortAfterEnqueue := errors.New("test: abort after Redis accepted the task")

			created, err := NewStore(pool).RegisterTx(ctx, RegisterTxParams{
				Email: email, PasswordHash: "test-hash",
				AfterCreate: func(account Account) error {
					inserted = account
					payload, err := json.Marshal(struct {
						AccountID int64 `json:"account_id"`
					}{AccountID: account.ID})
					if err != nil {
						return err
					}
					info, err := client.EnqueueContext(ctx,
						asynq.NewTask("task:send_verify_email", payload),
						asynq.Queue(queue), asynq.ProcessIn(time.Hour))
					if err != nil {
						return err
					}
					taskID = info.ID
					if tc.abort {
						return abortAfterEnqueue
					}
					return nil
				},
			})
			if tc.abort {
				require.ErrorIs(t, err, abortAfterEnqueue)
				require.Equal(t, inserted.ID, created.ID,
					"RegisterTx can return an uncommitted generated account ID together with an error")
			} else {
				require.NoError(t, err)
				require.Equal(t, inserted.ID, created.ID)
			}
			require.NotZero(t, inserted.ID, "AfterCreate was not reached")
			require.NotEmpty(t, taskID, "Redis enqueue did not succeed")

			// The queue task remains even when the PostgreSQL account, person
			// and party have all rolled back. The task is intentionally
			// scheduled far ahead, avoiding a worker race during this test.
			taskInfo, err := inspector.GetTaskInfo(queue, taskID)
			require.NoError(t, err, "Redis task should exist independently of PostgreSQL COMMIT")
			require.Equal(t, "task:send_verify_email", taskInfo.Type)
			var taskPayload struct {
				AccountID int64 `json:"account_id"`
			}
			require.NoError(t, json.Unmarshal(taskInfo.Payload, &taskPayload))
			require.Equal(t, inserted.ID, taskPayload.AccountID)

			for _, query := range []string{
				"SELECT count(*) FROM accounts WHERE id=$1",
				"SELECT count(*) FROM persons WHERE party_id=$1",
				"SELECT count(*) FROM parties WHERE id=$1",
			} {
				var count int
				id := inserted.ID
				if strings.Contains(query, "persons") || strings.Contains(query, "parties") {
					id = inserted.PersonID
				}
				require.NoError(t, pool.QueryRow(ctx, query, id).Scan(&count))
				require.Equal(t, tc.expected, count, query)
			}
		})
	}
}
