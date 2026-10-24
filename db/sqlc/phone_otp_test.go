package db

import (
    "bytes"
    "context"
    "errors"
    "testing"
    "strings"

    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
)

func TestPhoneOTPBindLoginAndReplay(t *testing.T) {
    pool:=openTestDB(t)
    ctx:=context.Background()
    store:=NewStore(pool)
    account,err:=store.RegisterTx(ctx,RegisterTxParams{Email:uuid.NewString()+"@example.test",PasswordHash:"hash"})
    require.NoError(t,err)
    phone:="+84"+randomPhoneSuffix(t)
    repo:=NewPhoneOTPStore(pool)
    good:=bytes.Repeat([]byte{1},32)
    bad:=bytes.Repeat([]byte{2},32)

    require.NoError(t,repo.StartChallenge(ctx,phone,"bind",account.ID,good,"123456"))
    require.ErrorIs(t,repo.StartChallenge(ctx,phone,"bind",account.ID,good,"123456"),ErrOTPCooldown)
    require.ErrorIs(t,repo.ConsumeChallenge(ctx,phone,"bind",account.ID,bad),ErrOTPInvalid)
    require.NoError(t,repo.ConsumeChallenge(ctx,phone,"bind",account.ID,good))
    require.ErrorIs(t,repo.ConsumeChallenge(ctx,phone,"bind",account.ID,good),ErrOTPInvalid)
    linked,err:=repo.AccountIDForPhone(ctx,phone)
    require.NoError(t,err)
    require.Equal(t,account.ID,linked)

    require.NoError(t,repo.StartChallenge(ctx,phone,"login",account.ID,good,"456789"))
    require.ErrorIs(t,repo.ConsumeChallenge(ctx,phone,"login",account.ID+1,good),ErrOTPInvalid)
    require.NoError(t,repo.ConsumeChallenge(ctx,phone,"login",account.ID,good))
    require.ErrorIs(t,repo.ConsumeChallenge(ctx,phone,"login",account.ID,good),ErrOTPInvalid)
}

func TestPhoneOTPAttemptLockAndUniqueness(t *testing.T) {
    pool:=openTestDB(t)
    ctx:=context.Background()
    store:=NewStore(pool)
    a,err:=store.RegisterTx(ctx,RegisterTxParams{Email:uuid.NewString()+"@example.test",PasswordHash:"hash"})
    require.NoError(t,err)
    b,err:=store.RegisterTx(ctx,RegisterTxParams{Email:uuid.NewString()+"@example.test",PasswordHash:"hash"})
    require.NoError(t,err)
    phone:="+84"+randomPhoneSuffix(t)
    repo:=NewPhoneOTPStore(pool)
    good:=bytes.Repeat([]byte{3},32)
    bad:=bytes.Repeat([]byte{4},32)
    require.NoError(t,repo.StartChallenge(ctx,phone,"bind",a.ID,good,"123456"))
    for i:=0;i<5;i++ {require.ErrorIs(t,repo.ConsumeChallenge(ctx,phone,"bind",a.ID,bad),ErrOTPInvalid)}
    _,err=pool.Exec(ctx,`UPDATE phone_otp_challenges
        SET next_send_at=now()-interval '1 second'
        WHERE phone_e164=$1 AND purpose='bind'`,phone)
    require.NoError(t,err)
    require.ErrorIs(t,repo.StartChallenge(ctx,phone,"bind",a.ID,good,"999999"),ErrOTPCooldown)
    require.ErrorIs(t,repo.ConsumeChallenge(ctx,phone,"bind",a.ID,good),ErrOTPInvalid)
    _,err=pool.Exec(ctx,`UPDATE phone_otp_challenges
        SET attempt_window_started_at=now()-interval '16 minutes',locked_until=now()-interval '1 second'
        WHERE phone_e164=$1 AND purpose='bind'`,phone)
    require.NoError(t,err)
    require.NoError(t,repo.StartChallenge(ctx,phone,"bind",a.ID,good,"999999"))
    require.NoError(t,repo.ConsumeChallenge(ctx,phone,"bind",a.ID,good))

    require.NoError(t,repo.StartChallenge(ctx,phone,"bind",b.ID,good,"555555"))
    err=repo.ConsumeChallenge(ctx,phone,"bind",b.ID,good)
    require.Error(t,err)
    require.False(t,errors.Is(err,ErrOTPInvalid))
    linked,err:=repo.AccountIDForPhone(ctx,phone)
    require.NoError(t,err)
    require.Equal(t,a.ID,linked)
}

func randomPhoneSuffix(t *testing.T) string {
    t.Helper()
    raw:=uuid.New()
    var b strings.Builder
    for _,v:=range raw { b.WriteByte('0'+v%10) }
    return b.String()[:10]
}
