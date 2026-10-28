package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestVerifyEmailHandlerRejectsBadPayload(t *testing.T) {
	h := NewVerifyEmailHandler(nil, nil, "")
	for _, payload := range [][]byte{[]byte("{"), []byte(`{"account_id":0}`)} {
		err := h.ProcessTask(context.Background(), asynq.NewTask(TaskSendVerifyEmail, payload))
		if !errors.Is(err, asynq.SkipRetry) {
			t.Fatalf("payload=%q err=%v", payload, err)
		}
	}
}
