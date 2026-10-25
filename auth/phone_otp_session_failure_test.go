package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type oneUseLoginChallenge struct {
	accountID int64
	consumed bool
	calls int
}

func (c *oneUseLoginChallenge) AccountIDForPhone(_ context.Context, _ string) (int64, error) {
	return c.accountID, nil
}
func (c *oneUseLoginChallenge) StartChallenge(_ context.Context, _, _ string, _ int64, _ []byte, _ string) error {
	return nil
}
func (c *oneUseLoginChallenge) ConsumeChallenge(_ context.Context, _, purpose string, accountID int64, _ []byte) error {
	c.calls++
	if purpose != "login" || accountID != c.accountID || c.consumed {
		return db.ErrOTPInvalid
	}
	c.consumed = true
	return nil
}

// Current policy: a valid OTP is consumed BEFORE CreateSession. A persistence
// outage returns no tokens, but does not re-enable an already consumed OTP.
// The fake models that one-use contract; it does not prove PG/OTP atomicity.
func TestPhoneOTPLoginPersistenceFailureDoesNotRestoreConsumedCode(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	challenge := &oneUseLoginChallenge{accountID: 31}
	cfg := util.Config{
		Environment: "development",
		TokenSymmetricKey: strings.Repeat("s", 32),
		AccessTokenDuration: 15 * time.Minute,
		RefreshTokenDuration: time.Hour,
	}
	maker, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
	require.NoError(t, err)
	account := db.Account{ID: 31, Email: "phone@example.test"}
	store.EXPECT().GetAccountByID(ctx, account.ID).Return(account, nil)
	outage := errors.New("postgres session insert unavailable")
	store.EXPECT().CreateSession(ctx, gomock.Any()).Return(db.Session{}, outage)
	service := PhoneOTP{Store: store, Challenges: challenge, Maker: maker, Config: cfg}

	first, err := service.VerifyLogin(ctx, "+84901234567", "123456", "agent", "ip")
	require.ErrorIs(t, err, outage)
	require.Equal(t, LoginResult{}, first)
	require.True(t, challenge.consumed)
	require.Equal(t, 1, challenge.calls)

	// Replay is rejected before account lookup or session issuance.
	replay, err := service.VerifyLogin(ctx, "+84901234567", "123456", "agent", "ip")
	require.ErrorIs(t, err, db.ErrOTPInvalid)
	require.Equal(t, LoginResult{}, replay)
	require.Equal(t, 2, challenge.calls)
}
