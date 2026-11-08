package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestPasswordResetWorkerFailsClosedForLegacyUnversionedTask(t *testing.T) {
	// A pre-migration payload has no generation; guessing zero could reopen
	// recovery after a password reset performed before versioning existed.
	handler := NewPasswordResetHandler(nil, nil, "")
	legacy := asynq.NewTask(TaskSendPasswordReset, []byte(`{"account_id":42}`))
	if err := handler.ProcessTask(context.Background(), legacy); err != nil {
		t.Fatalf("legacy task must be safely acknowledged without proof issuance: %v", err)
	}
}

func TestPasswordResetWorkerRejectsInvalidGeneration(t *testing.T) {
	handler := NewPasswordResetHandler(nil, nil, "")
	for _, payload := range [][]byte{
		[]byte("{"),
		[]byte(`{"account_id":0,"reset_version":0}`),
		[]byte(`{"account_id":42,"reset_version":-1}`),
	} {
		err := handler.ProcessTask(context.Background(), asynq.NewTask(TaskSendPasswordReset, payload))
		if !errors.Is(err, asynq.SkipRetry) {
			t.Fatalf("invalid payload %s must not retry: %v", payload, err)
		}
	}
}
