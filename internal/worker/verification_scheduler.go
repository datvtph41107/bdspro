package worker

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
)

// VerificationScheduler adapts the worker's queue contract to the registration
// operation. Queue and retry policy stay with the execution infrastructure.
type VerificationScheduler struct {
	distributor TaskDistributor
}

func NewVerificationScheduler(distributor TaskDistributor) VerificationScheduler {
	return VerificationScheduler{distributor: distributor}
}

func (s VerificationScheduler) ScheduleVerifyEmail(ctx context.Context, accountID int64) error {
	return s.distributor.DistributeTaskSendVerifyEmail(ctx,
		&PayloadSendVerifyEmail{AccountID: accountID},
		asynq.MaxRetry(10), asynq.ProcessIn(10*time.Second), asynq.Queue(QueueCritical))
}
