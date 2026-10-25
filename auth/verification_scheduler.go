package auth

import "context"

// VerificationTaskScheduler describes the verification intent emitted by
// registration. Auth owns *when* this action is required; the runtime adapter
// owns queue selection, retries, scheduling, and delivery semantics.
type VerificationTaskScheduler interface {
	ScheduleVerifyEmail(ctx context.Context, accountID int64) error
}
