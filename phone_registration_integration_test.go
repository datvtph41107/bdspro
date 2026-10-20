package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/datvtph41107/bdspro/auth"
)

func phoneRegistrationFixture(t *testing.T) (*auth.Service, *fakePhoneSMS) {
	t.Helper()
	db := coreDB(t)
	service := auth.New(db, nil)
	sms := &fakePhoneSMS{}
	if err := service.ConfigurePhoneVerification(strings.Repeat("R", 32), sms); err != nil {
		t.Fatal(err)
	}
	return service, sms
}

func TestPhoneOnlyRegistrationAtomicIdentityAndExistingLogin(t *testing.T) {
	a, sms := phoneRegistrationFixture(t)
	ctx := context.Background()
	const phone = "+84901112233"
	start, err := a.StartPhoneRegistration(ctx, phone)
	if err != nil || len(start.ChallengeToken) != 43 {
		t.Fatalf("start: %+v %v", start, err)
	}
	_, code, sends := sms.last()
	if sends != 1 || len(code) != 6 {
		t.Fatal("SMS was not delivered to dev transport")
	}
	var partiesBefore, accountsBefore, sessionsBefore int64
	for _, check := range []struct{ sql string; out *int64 }{
		{"SELECT count(*) FROM parties", &partiesBefore},
		{"SELECT count(*) FROM accounts", &accountsBefore},
		{"SELECT count(*) FROM sessions", &sessionsBefore},
	} {
		if err := a.Pool.QueryRow(ctx, check.sql).Scan(check.out); err != nil {
			t.Fatal(err)
		}
	}
	if partiesBefore != 0 || accountsBefore != 0 || sessionsBefore != 0 {
		t.Fatal("start created identity or session before OTP")
	}
	var mac string
	if err := a.Pool.QueryRow(ctx,
		"SELECT otp_mac FROM phone_registration_challenges WHERE challenge_hash=$1",
		phoneLoginHash(start.ChallengeToken)).Scan(&mac); err != nil {
		t.Fatal(err)
	}
	if len(mac) != 64 || strings.Contains(mac, code) {
		t.Fatal("plaintext registration OTP persisted in challenge table")
	}
	if _, err := a.VerifyPhoneRegistration(ctx, start.ChallengeToken, "broken"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("non-six-digit proof accepted: %v", err)
	}
	wrong := "000000"
	if wrong == code {
		wrong = "999999"
	}
	if _, err := a.VerifyPhoneRegistration(ctx, start.ChallengeToken, wrong); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("wrong OTP accepted: %v", err)
	}
	var attempts int32
	if err := a.Pool.QueryRow(ctx, "SELECT attempts FROM phone_registration_challenges WHERE challenge_hash=$1",
		phoneLoginHash(start.ChallengeToken)).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("wrong attempt rollback: %d %v", attempts, err)
	}
	reg, err := a.VerifyPhoneRegistration(ctx, start.ChallengeToken, code)
	if err != nil || reg.AccountID <= 0 || reg.Token == "" {
		t.Fatalf("registration verify: %+v %v", reg, err)
	}
	var personID int64
	if err := a.Pool.QueryRow(ctx, "SELECT person_id FROM accounts WHERE id=$1", reg.AccountID).Scan(&personID); err != nil || personID <= 0 {
		t.Fatalf("missing Person: %d %v", personID, err)
	}
	var verified bool
	if err := a.Pool.QueryRow(ctx, "SELECT verified_at IS NOT NULL FROM account_phones WHERE account_id=$1 AND phone=$2", reg.AccountID, phone).Scan(&verified); err != nil || !verified {
		t.Fatalf("verified phone missing: %v", err)
	}
	var emailCount, passwordCount, orgCount int
	for _, check := range []struct{ sql string; out *int }{
		{"SELECT count(*) FROM account_emails", &emailCount},
		{"SELECT count(*) FROM password_credentials", &passwordCount},
		{"SELECT count(*) FROM organization_memberships", &orgCount},
	} {
		if err := a.Pool.QueryRow(ctx, check.sql).Scan(check.out); err != nil {
			t.Fatal(err)
		}
	}
	if emailCount != 0 || passwordCount != 0 || orgCount != 0 {
		t.Fatal("phone-only Account unexpectedly created email/password or org membership")
	}
	if id, err := a.AccountFromToken(ctx, reg.Token); err != nil || id != reg.AccountID {
		t.Fatalf("new session not valid: id=%d err=%v", id, err)
	}
	if _, err := a.VerifyPhoneRegistration(ctx, start.ChallengeToken, code); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("replayed registration: %v", err)
	}
	again, err := a.StartPhoneRegistration(ctx, phone)
	if err != nil || len(again.ChallengeToken) != 43 {
		t.Fatalf("already-owned phone leaked account via start: %+v %v", again, err)
	}
	_, _, n := sms.last()
	if n != 1 {
		t.Fatal("existing verified phone received signup code")
	}
	if _, err := a.VerifyPhoneRegistration(ctx, again.ChallengeToken, code); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("fake handle authenticated: %v", err)
	}
	// The destination quota is shared across registration and login.
	if _, err := a.Pool.Exec(ctx, "UPDATE phone_registration_challenges SET created_at=now()-interval '61 seconds'"); err != nil {
		t.Fatal(err)
	}
	login, err := a.StartPhoneLogin(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	_, loginCode, n := sms.last()
	if n != 2 {
		t.Fatalf("existing phone login did not send after cooldown: %d", n)
	}
	session, err := a.VerifyPhoneLogin(ctx, login.ChallengeToken, loginCode)
	if err != nil || session.AccountID != reg.AccountID {
		t.Fatalf("same Account not reused: %d vs %d: %v", session.AccountID, reg.AccountID, err)
	}
}

func TestPhoneRegistrationConcurrentRedeemIsSingleUse(t *testing.T) {
	a, sms := phoneRegistrationFixture(t)
	ctx := context.Background()
	start, err := a.StartPhoneRegistration(ctx, "+84905556677")
	if err != nil {
		t.Fatal(err)
	}
	_, code, _ := sms.last()
	var wg sync.WaitGroup
	startAll := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startAll
			_, err := a.VerifyPhoneRegistration(ctx, start.ChallengeToken, code)
			results <- err
		}()
	}
	close(startAll)
	wg.Wait()
	close(results)
	success, invalid := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, auth.ErrInvalidCredentials) {
			invalid++
		} else {
			t.Fatalf("unexpected duplicate redemption error: %v", err)
		}
	}
	if success != 1 || invalid != 1 {
		t.Fatalf("one challenge minted multiple Accounts: success=%d invalid=%d", success, invalid)
	}
	var count int
	if err := a.Pool.QueryRow(ctx, "SELECT count(*) FROM accounts").Scan(&count); err != nil || count != 1 {
		t.Fatalf("unexpected Account rows: %d %v", count, err)
	}
}

func TestPhoneRegistrationAttemptLimitAndExpiry(t *testing.T) {
	a, sms := phoneRegistrationFixture(t)
	ctx := context.Background()
	start, err := a.StartPhoneRegistration(ctx, "+84907778899")
	if err != nil {
		t.Fatal(err)
	}
	_, code, _ := sms.last()
	wrong := "000000"
	if wrong == code {
		wrong = "999999"
	}
	for i := 0; i < 5; i++ {
		if _, err := a.VerifyPhoneRegistration(ctx, start.ChallengeToken, wrong); !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if _, err := a.VerifyPhoneRegistration(ctx, start.ChallengeToken, code); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("more than five guesses accepted: %v", err)
	}
	// Another phone's challenge is separately bounded by expiry.
	second, err := a.StartPhoneRegistration(ctx, "+84907778888")
	if err != nil {
		t.Fatal(err)
	}
	_, secondCode, _ := sms.last()
	if _, err := a.Pool.Exec(ctx,
		"UPDATE phone_registration_challenges SET created_at=now()-interval '10 minutes', expires_at=now()-interval '1 second' WHERE challenge_hash=$1",
		phoneLoginHash(second.ChallengeToken)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.VerifyPhoneRegistration(ctx, second.ChallengeToken, secondCode); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expired registration proof accepted: %v", err)
	}
	var count int
	if err := a.Pool.QueryRow(ctx, "SELECT count(*) FROM accounts").Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed proofs created Accounts: %d %v", count, err)
	}
}
