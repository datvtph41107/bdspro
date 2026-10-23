package worker

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

// Exercises Asynq's Redis-backed scheduling and retry, not just a mocked handler.
// CI provides REDIS_TEST_ADDRESS; ordinary offline unit tests skip this integration.
func TestAsynqRedisScheduleAndRetry(t *testing.T) {
	addr := os.Getenv("REDIS_TEST_ADDRESS")
	if addr == "" {
		t.Skip("REDIS_TEST_ADDRESS not set")
	}

	redisOpt := asynq.RedisClientOpt{Addr: addr}
	queueName := "bdspro_test_" + uuid.NewString()
	client := asynq.NewClient(redisOpt)
	defer client.Close()
	inspector := asynq.NewInspector(redisOpt)
	defer inspector.Close()

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 1,
		Queues: map[string]int{queueName: 1},
		DelayedTaskCheckInterval: 100 * time.Millisecond,
		RetryDelayFunc: func(int, error, *asynq.Task) time.Duration {
			return 100 * time.Millisecond
		},
	})
	var attempts atomic.Int32
	done := make(chan struct{})
	mux := asynq.NewServeMux()
	mux.HandleFunc("test:scheduled_retry", func(ctx context.Context, task *asynq.Task) error {
		call := attempts.Add(1)
		if call == 1 {
			return errors.New("transient mail failure")
		}
		retried, ok := asynq.GetRetryCount(ctx)
		if !ok || retried != 1 {
			return errors.New("retry count was not preserved in task context")
		}
		close(done)
		return nil
	})

	require.NoError(t, server.Start(mux))
	defer func() {
		server.Shutdown()
		require.NoError(t, inspector.DeleteQueue(queueName, true))
	}()

	_, err := client.EnqueueContext(context.Background(),
		asynq.NewTask("test:scheduled_retry", []byte(`{"account_id":25}`)),
		asynq.Queue(queueName), asynq.ProcessIn(200*time.Millisecond), asynq.MaxRetry(1),
	)
	require.NoError(t, err)

	select {
	case <-done:
		require.EqualValues(t, 2, attempts.Load())
	case <-time.After(12 * time.Second):
		t.Fatalf("Asynq did not schedule and retry task; attempts=%d", attempts.Load())
	}
}
