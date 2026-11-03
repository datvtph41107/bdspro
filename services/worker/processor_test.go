package worker

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"
)

func TestTaskRegistrationSupportsIndependentHandlers(t *testing.T) {
	p := NewRedisTaskProcessor(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"})
	var emailCalls, reportCalls int
	p.Handle("email.verify.test", asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		emailCalls++
		return nil
	}))
	p.Handle("report.generate.test", asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		reportCalls++
		return nil
	}))
	for _, name := range []string{"email.verify.test", "report.generate.test"} {
		if err := p.mux.ProcessTask(context.Background(), asynq.NewTask(name, nil)); err != nil {
			t.Fatal(err)
		}
	}
	if emailCalls != 1 || reportCalls != 1 {
		t.Fatalf("email=%d report=%d", emailCalls, reportCalls)
	}
}
