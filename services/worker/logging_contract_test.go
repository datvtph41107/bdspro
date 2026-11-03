package worker

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

func TestTaskFailureLogRedactsErrorAndPayload(t *testing.T) {
	var output bytes.Buffer
	logger := zerolog.New(&output).Level(zerolog.DebugLevel)
	ctx := logger.WithContext(context.Background())
	task := asynq.NewTask("task:send_verify_email", []byte("private-otp-payload"))

	logTaskFailure(ctx, task, errors.New("private-smtp-password"))

	if !strings.Contains(output.String(), `"message":"task.failed"`) ||
		!strings.Contains(output.String(), `"task":"task:send_verify_email"`) ||
		!strings.Contains(output.String(), `"error_type":`) {
		t.Fatalf("missing safe task diagnostics: %s", output.String())
	}
	if strings.Contains(output.String(), "private-smtp-password") ||
		strings.Contains(output.String(), "private-otp-payload") {
		t.Fatalf("worker logged private data: %s", output.String())
	}
}

func TestWorkerPrintfUsesProvidedContextLogger(t *testing.T) {
	var output bytes.Buffer
	logger := zerolog.New(&output).Level(zerolog.DebugLevel).
		With().Str("trace_id", "task-trace-1").Logger()

	NewLogger().Printf(logger.WithContext(context.Background()), "task status: %d", 2)

	if !strings.Contains(output.String(), `"trace_id":"task-trace-1"`) ||
		!strings.Contains(output.String(), "task status: 2") {
		t.Fatalf("worker logger ignored context: %s", output.String())
	}
}
