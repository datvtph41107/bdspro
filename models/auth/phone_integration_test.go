package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The real DB is required: a mock cannot prove that a wrong-code attempt
// commits while all identity writes remain in one PostgreSQL transaction.
func phoneTestStore(t *testing.T) (context.Context, *pgxpool.Pool, db.Store) {
	t.Helper()
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable bdspro_test PostgreSQL database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("BDS_TEST_MODE=disposable is mandatory")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing to write outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, pool, db.NewStore(pool)
}

func randomPhone(t *testing.T) string {
	t.Helper()
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		t.Fatal(err)
	}
	n := binary.BigEndian.Uint64(data[:])%900000000 + 100000000
	return fmt.Sprintf("+84%09d", n)
}

func provisionPhoneAccount(t *testing.T, ctx context.Context, store db.Store, phone string) db.Account {
	t.Helper()
	mac := sha256.Sum256([]byte("phone-register:"+phone))
	if err := auth_model.StartPhoneSignup(ctx, store, phone, mac[:], "123456"); err != nil {
		t.Fatal(err)
	}
	account, err := auth_model.CompletePhoneSignup(ctx, store, phone, mac[:])
	if err != nil {
		t.Fatal(err)
	}
	return account
}

func TestPhoneChallengeCommitsWrongCodeAndConsumesOnce(t *testing.T) {
	ctx, pool, store := phoneTestStore(t)
	phone := randomPhone(t)
	account := provisionPhoneAccount(t, ctx, store, phone)
	mac := sha256.Sum256([]byte("correct-login:"+phone))
	wrong := sha256.Sum256([]byte("wrong-login:"+phone))

	if err := auth_model.StartChallenge(ctx, store, phone, "login", account.ID, mac[:], "234567"); err != nil {
		t.Fatal(err)
	}
	if err := auth_model.ConsumeChallenge(ctx, store, phone, "login", account.ID, wrong[:]); !errors.Is(err, auth_model.ErrOTPInvalid) {
		t.Fatalf("wrong OTP accepted: %v", err)
	}
	var attempts int
	err := pool.QueryRow(ctx, "SELECT failed_attempts FROM phone_otp_challenges WHERE phone_e164=$1 AND purpose='login'", phone).Scan(&attempts)
	if err != nil || attempts != 1 {
		t.Fatalf("wrong OTP counter was not committed: attempts=%d err=%v", attempts, err)
	}
	if err := auth_model.ConsumeChallenge(ctx, store, phone, "login", account.ID, mac[:]); err != nil {
		t.Fatalf("valid OTP rejected: %v", err)
	}
	if err := auth_model.ConsumeChallenge(ctx, store, phone, "login", account.ID, mac[:]); !errors.Is(err, auth_model.ErrOTPInvalid) {
		t.Fatalf("OTP replay accepted: %v", err)
	}
}

func TestPhoneChallengeLockoutAfterFiveFailures(t *testing.T) {
	ctx, pool, store := phoneTestStore(t)
	phone := randomPhone(t)
	account := provisionPhoneAccount(t, ctx, store, phone)
	mac := sha256.Sum256([]byte("correct:"+phone))
	wrong := sha256.Sum256([]byte("wrong:"+phone))
	if err := auth_model.StartChallenge(ctx, store, phone, "login", account.ID, mac[:], "234567"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := auth_model.ConsumeChallenge(ctx, store, phone, "login", account.ID, wrong[:]); !errors.Is(err, auth_model.ErrOTPInvalid) {
			t.Fatalf("failure %d: %v", i+1, err)
		}
	}
	var attempts int
	var locked time.Time
	if err := pool.QueryRow(ctx, "SELECT failed_attempts, locked_until FROM phone_otp_challenges WHERE phone_e164=$1 AND purpose='login'", phone).Scan(&attempts, &locked); err != nil {
		t.Fatal(err)
	}
	if attempts != 5 || !locked.After(time.Now()) {
		t.Fatalf("wrong lockout state: attempts=%d locked=%v", attempts, locked)
	}
	if err := auth_model.ConsumeChallenge(ctx, store, phone, "login", account.ID, mac[:]); !errors.Is(err, auth_model.ErrOTPInvalid) {
		t.Fatalf("locked OTP accepted: %v", err)
	}
}

func TestConcurrentPhoneProofOnlyOneSuccess(t *testing.T) {
	ctx, _, store := phoneTestStore(t)
	phone := randomPhone(t)
	account := provisionPhoneAccount(t, ctx, store, phone)
	mac := sha256.Sum256([]byte("one-time:"+phone))
	if err := auth_model.StartChallenge(ctx, store, phone, "login", account.ID, mac[:], "234567"); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- auth_model.ConsumeChallenge(ctx, store, phone, "login", account.ID, mac[:])
		}()
	}
	wg.Wait()
	close(results)
	success, invalid := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, auth_model.ErrOTPInvalid):
			invalid++
		default:
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if success != 1 || invalid != 1 {
		t.Fatalf("proof used more than once: successes=%d invalids=%d", success, invalid)
	}
}
