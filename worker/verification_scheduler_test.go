package worker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/datvtph41107/bdspro/worker"
	mockwk "github.com/datvtph41107/bdspro/worker/mock"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestVerificationSchedulerAdaptsApplicationIntent(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	distributor := mockwk.NewMockTaskDistributor(ctrl)
	distributor.EXPECT().DistributeTaskSendVerifyEmail(
		ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).DoAndReturn(func(got context.Context, payload *worker.PayloadSendVerifyEmail, opts ...asynq.Option) error {
		require.Same(t, ctx, got)
		require.Equal(t, int64(81), payload.AccountID)
		require.Len(t, opts, 3, "keep retry, delay, and queue selection under worker ownership")
		return nil
	})
	require.NoError(t, worker.NewVerificationScheduler(distributor).ScheduleVerifyEmail(ctx, 81))
}

func TestVerificationSchedulerPreservesEnqueueError(t *testing.T) {
	ctrl := gomock.NewController(t)
	distributor := mockwk.NewMockTaskDistributor(ctrl)
	failure := errors.New("redis unavailable")
	distributor.EXPECT().DistributeTaskSendVerifyEmail(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(failure)
	err := worker.NewVerificationScheduler(distributor).ScheduleVerifyEmail(context.Background(), 81)
	require.ErrorIs(t, err, failure)
}
