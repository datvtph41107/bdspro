package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/token"
)

type fakePhoneSMS struct {
	mu       sync.Mutex
	to, code string
	count    int
	err      error
}

func (s *fakePhoneSMS) SendOTP(_ context.Context, to, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.to = to
	s.code = code
	s.count++
	return s.err
}

func (s *fakePhoneSMS) last() (string, string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.to, s.code, s.count
}

func TestPhoneOTPSingleUseCooldownAndAttempts(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	authService := auth.New(db, nil)
	owner, _ := coreUser(t, authService, "otp-owner@example.test")
	sender := &fakePhoneSMS{}
	if err := authService.ConfigurePhoneVerification(strings.Repeat("X", 32), sender); err != nil {
		t.Fatal(err)
	}
	phone := "+84901234567"
	if err := authService.StartPhoneVerification(ctx, owner, phone); err != nil {
		t.Fatal(err)
	}
	to, code, count := sender.last()
	if to != phone || len(code) != 6 || count != 1 {
		t.Fatalf("sender mismatch: to=%q code-len=%d count=%d", to, len(code), count)
	}
	var stored string
	if err := db.QueryRow(ctx, "SELECT otp_mac FROM phone_verification_challenges ORDER BY id DESC LIMIT 1").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, code) || len(stored) != 64 {
		t.Fatal("OTP exposed in database")
	}
	if err := authService.StartPhoneVerification(ctx, owner, phone); !errors.Is(err, auth.ErrPhoneThrottle) {
		t.Fatalf("cooldown: %v", err)
	}
	wrong := "000000"
	if wrong == code {
		wrong = "999999"
	}
	if err := authService.VerifyPhoneCode(ctx, owner, phone, wrong); !errors.Is(err, auth.ErrPhoneCode) {
		t.Fatalf("wrong guess: %v", err)
	}
	var attempts int
	if err := db.QueryRow(ctx, "SELECT attempts FROM phone_verification_challenges ORDER BY id DESC LIMIT 1").Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("failed guess did not persist: %d", attempts)
	}
	if err := authService.VerifyPhoneCode(ctx, owner, phone, code); err != nil {
		t.Fatalf("correct OTP rejected: %v", err)
	}
	var verified bool
	if err := db.QueryRow(ctx, "SELECT verified_at IS NOT NULL FROM account_phones WHERE account_id=$1 AND phone=$2", owner, phone).Scan(&verified); err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("phone not verified")
	}
	if err := authService.VerifyPhoneCode(ctx, owner, phone, code); !errors.Is(err, auth.ErrPhoneCode) {
		t.Fatalf("single-use violation: %v", err)
	}
}

func TestPhoneOTPRejectsFifthBadAttempt(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	id, _ := coreUser(t, a, "attempts@example.test")
	sms := &fakePhoneSMS{}
	if err := a.ConfigurePhoneVerification(strings.Repeat("Y", 32), sms); err != nil {
		t.Fatal(err)
	}
	const phone = "+84909998877"
	if err := a.StartPhoneVerification(ctx, id, phone); err != nil {
		t.Fatal(err)
	}
	_, code, _ := sms.last()
	wrong := "000000"
	if wrong == code {
		wrong = "999999"
	}
	for i := 0; i < 5; i++ {
		if err := a.VerifyPhoneCode(ctx, id, phone, wrong); !errors.Is(err, auth.ErrPhoneCode) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if err := a.VerifyPhoneCode(ctx, id, phone, code); !errors.Is(err, auth.ErrPhoneCode) {
		t.Fatalf("sixth attempt accepted: %v", err)
	}
	var attempts int
	if err := db.QueryRow(ctx, "SELECT attempts FROM phone_verification_challenges ORDER BY id DESC LIMIT 1").Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 5 {
		t.Fatalf("attempts=%d", attempts)
	}
}

func TestPhoneOTPVerifiedNumberExclusiveAcrossAccounts(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	id1, _ := coreUser(t, a, "number1@example.test")
	id2, _ := coreUser(t, a, "number2@example.test")
	sms := &fakePhoneSMS{}
	if err := a.ConfigurePhoneVerification(strings.Repeat("Q", 32), sms); err != nil {
		t.Fatal(err)
	}
	const phone = "+84901230000"
	if err := a.StartPhoneVerification(ctx, id1, phone); err != nil {
		t.Fatal(err)
	}
	_, code, _ := sms.last()
	if err := a.VerifyPhoneCode(ctx, id1, phone, code); err != nil {
		t.Fatal(err)
	}
	// Permit second issuance after the global destination cooldown to test
	// the partial unique constraint instead of the SMS abuse limiter.
	if _, err := db.Exec(ctx, `UPDATE phone_verification_challenges
		SET created_at = now() - interval '61 seconds'
		WHERE account_phone_id IN (SELECT id FROM account_phones WHERE phone=$1)`, phone); err != nil {
		t.Fatal(err)
	}
	if err := a.StartPhoneVerification(ctx, id2, phone); err != nil {
		t.Fatal(err)
	}
	_, second, _ := sms.last()
	if err := a.VerifyPhoneCode(ctx, id2, phone, second); !errors.Is(err, auth.ErrPhoneInUse) {
		t.Fatalf("verified number can be claimed twice: %v", err)
	}
}

func TestJWTAccessTracksActualSessionRevocation(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	id, opaque := coreUser(t, a, "jwtowner@example.test")
	issuer, err := token.NewJWTMaker(strings.Repeat("J", 32))
	if err != nil {
		t.Fatal(err)
	}
	jwtToken, _, err := issuer.Issue(id, opaque)
	if err != nil {
		t.Fatal(err)
	}
	accountID, sessionHash, err := issuer.Verify(jwtToken)
	if err != nil || accountID != id {
		t.Fatalf("jwt verify: %v", err)
	}
	verified, err := a.AccountFromSessionHash(ctx, sessionHash)
	if err != nil || verified != id {
		t.Fatalf("session binding: %v", err)
	}
	if err := a.LogoutAll(ctx, opaque); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AccountFromSessionHash(ctx, sessionHash); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("revoked session accepted via JWT: %v", err)
	}
}

func TestPhoneOTPAccountWideDailyQuotaAcrossNumbers(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	id, _ := coreUser(t, a, "accountwide@example.test")
	sms := &fakePhoneSMS{}
	if err := a.ConfigurePhoneVerification(strings.Repeat("M", 32), sms); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		phone := fmt.Sprintf("+84908888%03d", i)
		if err := a.StartPhoneVerification(ctx, id, phone); err != nil {
			t.Fatalf("issuance %d: %v", i, err)
		}
	}
	if err := a.StartPhoneVerification(ctx, id, "+84908888999"); !errors.Is(err, auth.ErrPhoneThrottle) {
		t.Fatalf("account-wide SMS cap bypassed: %v", err)
	}
	_, _, sent := sms.last()
	if sent != 10 {
		t.Fatalf("SMS count %d, expected 10", sent)
	}
}

func TestPhoneOTPGlobalDestinationCooldownAcrossAccounts(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	first, _ := coreUser(t, a, "globalphone1@example.test")
	second, _ := coreUser(t, a, "globalphone2@example.test")
	sms := &fakePhoneSMS{}
	if err := a.ConfigurePhoneVerification(strings.Repeat("G", 32), sms); err != nil {
		t.Fatal(err)
	}
	const phone = "+84907770123"
	if err := a.StartPhoneVerification(ctx, first, phone); err != nil {
		t.Fatal(err)
	}
	if err := a.StartPhoneVerification(ctx, second, phone); !errors.Is(err, auth.ErrPhoneThrottle) {
		t.Fatalf("cross-account destination limiter bypassed: %v", err)
	}
	_, _, sent := sms.last()
	if sent != 1 {
		t.Fatalf("expected one SMS, sent %d", sent)
	}
}

func TestPhoneOTPReissueInvalidatesPriorCode(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	id, _ := coreUser(t, a, "otp-reissue@example.test")
	sms := &fakePhoneSMS{}
	if err := a.ConfigurePhoneVerification(strings.Repeat("R", 32), sms); err != nil {
		t.Fatal(err)
	}
	const phone = "+84903331234"
	if err := a.StartPhoneVerification(ctx, id, phone); err != nil {
		t.Fatal(err)
	}
	_, firstCode, _ := sms.last()
	if _, err := db.Exec(ctx, `UPDATE phone_verification_challenges
		SET created_at = now() - interval '61 seconds'
		WHERE account_phone_id IN (SELECT id FROM account_phones WHERE account_id=$1 AND phone=$2)`, id, phone); err != nil {
		t.Fatal(err)
	}
	if err := a.StartPhoneVerification(ctx, id, phone); err != nil {
		t.Fatal(err)
	}
	_, newCode, sent := sms.last()
	if sent != 2 {
		t.Fatalf("expected second OTP SMS, got %d", sent)
	}
	if firstCode == newCode {
		// A one-in-a-million collision is allowed by the random generator, so
		// rely on database invalidation rather than value inequality.
		var priorConsumed bool
		if err := db.QueryRow(ctx, `SELECT consumed_at IS NOT NULL FROM
			phone_verification_challenges ORDER BY id LIMIT 1`).Scan(&priorConsumed); err != nil {
			t.Fatal(err)
		}
		if !priorConsumed {
			t.Fatal("previous OTP was not consumed on reissue")
		}
	} else if err := a.VerifyPhoneCode(ctx, id, phone, firstCode); !errors.Is(err, auth.ErrPhoneCode) {
		t.Fatalf("prior OTP accepted after replacement: %v", err)
	}
	if err := a.VerifyPhoneCode(ctx, id, phone, newCode); err != nil {
		t.Fatalf("new OTP rejected: %v", err)
	}
}

func TestPhoneOTPGlobalDestinationQuotaConcurrentAccounts(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	first, _ := coreUser(t, a, "concurrentphone1@example.test")
	second, _ := coreUser(t, a, "concurrentphone2@example.test")
	sms := &fakePhoneSMS{}
	if err := a.ConfigurePhoneVerification(strings.Repeat("C", 32), sms); err != nil {
		t.Fatal(err)
	}
	const phone = "+84906660123"
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, accountID := range []int64{first, second} {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			<-start
			results <- a.StartPhoneVerification(ctx, id, phone)
		}(accountID)
	}
	close(start)
	wg.Wait()
	close(results)
	accepted, throttled := 0, 0
	for err := range results {
		switch {
		case err == nil:
			accepted++
		case errors.Is(err, auth.ErrPhoneThrottle):
			throttled++
		default:
			t.Fatalf("unexpected concurrent OTP result: %v", err)
		}
	}
	if accepted != 1 || throttled != 1 {
		t.Fatalf("global phone quota bypassed: accepted=%d throttled=%d", accepted, throttled)
	}
	_, _, sent := sms.last()
	if sent != 1 {
		t.Fatalf("expected exactly one SMS, got %d", sent)
	}
}
