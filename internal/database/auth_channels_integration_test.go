package db

import (
    "context"
    "crypto/sha256"
    "errors"
    "fmt"
    "os"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

func authChannelsPool(t *testing.T) (*pgxpool.Pool, context.Context) {
    t.Helper()
    source := os.Getenv("BDS_TEST_DB_URL")
    if source == "" { t.Skip("requires disposable PostgreSQL") }
    if os.Getenv("BDS_TEST_MODE") != "disposable" { t.Fatal("write tests require disposable mode") }
    settings, err := pgxpool.ParseConfig(source)
    if err != nil { t.Fatal(err) }
    if settings.ConnConfig.Database != "bdspro_test" { t.Fatal("refusing writes to non-test database") }
    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    t.Cleanup(cancel)
    pool, err := pgxpool.NewWithConfig(ctx, settings)
    if err != nil { t.Fatal(err) }
    t.Cleanup(pool.Close)
    if err := pool.Ping(ctx); err != nil { t.Fatal(err) }
    return pool, ctx
}

func TestPhoneOnlySignupIsAtomicAndUnique(t *testing.T) {
    pool, ctx := authChannelsPool(t)
    store := NewStore(pool)
    phone := "+849" + fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
    correct := sha256.Sum256([]byte(uuid.NewString()))
    incorrect := sha256.Sum256([]byte(uuid.NewString()))
    if err := store.StartPhoneSignup(ctx, phone, correct[:], "123456"); err != nil { t.Fatal(err) }
    if _, err := store.CompletePhoneSignup(ctx, phone, incorrect[:]); !errors.Is(err, ErrOTPInvalid) {
        t.Fatalf("wrong OTP must fail: %v", err)
    }
    var attempts int
    if err := pool.QueryRow(ctx, "SELECT failed_attempts FROM phone_signup_challenges WHERE phone_e164=$1", phone).Scan(&attempts); err != nil { t.Fatal(err) }
    if attempts != 1 { t.Fatalf("wrong OTP attempts must commit: %d", attempts) }
    account, err := store.CompletePhoneSignup(ctx, phone, correct[:])
    if err != nil { t.Fatal(err) }
    if account.ID <= 0 || account.Email != "" || account.IsEmailVerified {
        t.Fatalf("phone-only account must have no invented email: %+v", account)
    }
    fromDB, err := store.GetAccountByID(ctx, account.ID)
    if err != nil || fromDB.Email != "" { t.Fatalf("nullable email read failed: %+v %v", fromDB, err) }
    var isNull bool
    if err := pool.QueryRow(ctx, "SELECT email IS NULL FROM accounts WHERE id=$1", account.ID).Scan(&isNull); err != nil { t.Fatal(err) }
    if !isNull { t.Fatal("phone-only account email must be SQL NULL") }
    if err := store.StartPhoneSignup(ctx, phone, correct[:], "123456"); err != nil { t.Fatal(err) }
    // The service checks availability, but a concurrent bypass must still
    // fail against the unique phone credential constraint in PostgreSQL.
    _, err = store.CompletePhoneSignup(ctx, phone, correct[:])
    var pgErr *pgconn.PgError
    if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
        t.Fatalf("duplicate verified phone must violate uniqueness: %v", err)
    }
    owner, err := store.AccountIDForPhone(ctx, phone)
    if err != nil || owner != account.ID { t.Fatalf("original phone ownership changed: %d %v", owner, err) }
}

func TestPasswordResetSingleUseAndRevokesRefreshSessions(t *testing.T) {
    pool, ctx := authChannelsPool(t)
    store := NewStore(pool)
    account, err := store.CreateGoogleAccountTx(ctx, uuid.NewString()+"@example.test", uuid.NewString())
    if err != nil { t.Fatal(err) }
    if _, err := store.GetPasswordHash(ctx, account.ID); !errors.Is(err, pgx.ErrNoRows) {
        t.Fatalf("Google account must start passwordless: %v", err)
    }
    oldSessionID := uuid.New()
    if _, err := store.CreateSession(ctx, CreateSessionParams{
        ID: oldSessionID, AccountID: account.ID, RefreshToken: "test-refresh", UserAgent: "integration-test",
        ClientIp: "127.0.0.1", ExpiresAt: time.Now().Add(time.Hour),
    }); err != nil { t.Fatal(err) }
    first := sha256.Sum256([]byte(uuid.NewString()))
    second := sha256.Sum256([]byte(uuid.NewString()))
    for _, hash := range [][]byte{first[:], second[:]} {
        if err := store.CreatePasswordResetToken(ctx, CreatePasswordResetTokenParams{TokenHash: hash, AccountID: account.ID}); err != nil { t.Fatal(err) }
    }
    if err := store.ResetPasswordTx(ctx, first[:], "test-new-hash"); err != nil { t.Fatal(err) }
    hash, err := store.GetPasswordHash(ctx, account.ID)
    if err != nil || hash != "test-new-hash" { t.Fatalf("new password not saved: %q %v", hash, err) }
    if _, err := store.GetSession(ctx, oldSessionID); !errors.Is(err, pgx.ErrNoRows) {
        t.Fatalf("old refresh session survived password change: %v", err)
    }
    for _, token := range [][]byte{first[:], second[:]} {
        if err := store.ResetPasswordTx(ctx, token, "replay"); !errors.Is(err, ErrPasswordResetInvalid) {
            t.Fatalf("reset token replay must fail: %v", err)
        }
    }
}
