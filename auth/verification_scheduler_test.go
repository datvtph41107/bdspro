package auth

import (
	"context"
	"errors"
	"testing"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type verificationSchedulerFunc func(context.Context, int64) error

func (f verificationSchedulerFunc) ScheduleVerifyEmail(ctx context.Context, accountID int64) error {
	return f(ctx, accountID)
}

type registrationContextKey struct{}

func TestRegisterSchedulesVerificationThroughApplicationPort(t *testing.T) {
	ctx := context.WithValue(context.Background(), registrationContextKey{}, "request")
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	expected := db.Account{ID: 73, PersonID: 21, Email: "first@example.test"}
	calls := 0
	scheduler := verificationSchedulerFunc(func(received context.Context, accountID int64) error {
		calls++
		require.Same(t, ctx, received)
		require.Equal(t, expected.ID, accountID)
		return nil
	})
	store.EXPECT().RegisterTx(ctx, gomock.Any()).
		DoAndReturn(func(received context.Context, params db.RegisterTxParams) (db.Account, error) {
			require.Same(t, ctx, received)
			require.Equal(t, expected.Email, params.Email)
			require.NoError(t, util.CheckPassword("strong-password", params.PasswordHash))
			require.NotNil(t, params.AfterCreate)
			return expected, params.AfterCreate(expected)
		})
	got, err := Register(ctx, store, scheduler, "  FIRST@example.test  ", "strong-password")
	require.NoError(t, err)
	require.Equal(t, expected, got)
	require.Equal(t, 1, calls)
}

func TestRegisterPropagatesSchedulingFailureToTransaction(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	redisFailure := errors.New("verification enqueue unavailable")
	scheduler := verificationSchedulerFunc(func(_ context.Context, id int64) error {
		require.Equal(t, int64(73), id)
		return redisFailure
	})
	store.EXPECT().RegisterTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, params db.RegisterTxParams) (db.Account, error) {
			return db.Account{}, params.AfterCreate(db.Account{ID: 73})
		})
	_, err := Register(ctx, store, scheduler, "user@example.test", "strong-password")
	require.ErrorIs(t, err, redisFailure)
}
