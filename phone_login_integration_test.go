package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/jackc/pgx/v5/pgxpool"
)

func phoneLoginFixture(t *testing.T) (*pgxpool.Pool, *auth.Service, *fakePhoneSMS, int64, string) {
	t.Helper()
	db := coreDB(t)
	a := auth.New(db, nil)
	accountID, _ := coreUser(t, a, "phone-login@example.test")
	sms := &fakePhoneSMS{}
	if err := a.ConfigurePhoneVerification(strings.Repeat("P", 32), sms); err != nil {
		t.Fatal(err)
	}
	const phone = "+84901234888"
	if err := a.StartPhoneVerification(context.Background(), accountID, phone); err != nil {
		t.Fatal(err)
	}
	_, enrollmentCode, _ := sms.last()
	if err := a.VerifyPhoneCode(context.Background(), accountID, phone, enrollmentCode); err != nil {
		t.Fatal(err)
	}
	// The enrollment SMS and anonymous login share a 60s destination budget.
	// Move issuance time only in this disposable integration test database.
	if _, err := db.Exec(context.Background(), "UPDATE phone_verification_challenges SET created_at=now()-interval '61 seconds'"); err != nil {
		t.Fatal(err)
	}
	return db, a, sms, accountID, phone
}

func TestPhoneLoginDoesNotEnumerateAndCannotCreateSessionBeforeProof(t *testing.T) {
	db, a, sms, accountID, phone := phoneLoginFixture(t)
	ctx := context.Background()
	var before int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE account_id=$1", accountID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	unknown, err := a.StartPhoneLogin(ctx, "+84908887766")
	if err != nil || len(unknown.ChallengeToken) != 43 {
		t.Fatalf("unknown phone start: %+v %v", unknown, err)
	}
	_, _, sendsBefore := sms.last()
	start, err := a.StartPhoneLogin(ctx, phone)
	if err != nil || len(start.ChallengeToken) != len(unknown.ChallengeToken) {
		t.Fatalf("verified phone start: %+v %v", start, err)
	}
	_, code, sendsAfter := sms.last()
	if sendsAfter != sendsBefore+1 {
		t.Fatalf("wrong SMS count before=%d after=%d", sendsBefore, sendsAfter)
	}
	if _, err := a.VerifyPhoneLogin(ctx, unknown.ChallengeToken, code); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("unknown handle authenticated: %v", err)
	}
	var after int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE account_id=$1", accountID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("starting a phone login or verifying a fake handle created a session")
	}
	result, err := a.VerifyPhoneLogin(ctx, start.ChallengeToken, code)
	if err != nil || result.AccountID != accountID || result.Token == "" {
		t.Fatalf("verified login failed: %+v %v", result, err)
	}
	if id, err := a.AccountFromToken(ctx, result.Token); err != nil || id != accountID {
		t.Fatalf("phone login session invalid: id=%d err=%v", id, err)
	}
	if _, err := a.VerifyPhoneLogin(ctx, start.ChallengeToken, code); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("replayed phone login OTP: %v", err)
	}
	if err := a.Logout(ctx, result.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AccountFromToken(ctx, result.Token); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("logout did not revoke phone session: %v", err)
	}
}

func TestPhoneLoginAttemptQuotaExpiryAndVerifiedStatus(t *testing.T) {
	db, a, sms, accountID, phone := phoneLoginFixture(t)
	ctx := context.Background()
	start, err := a.StartPhoneLogin(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	_, code, _ := sms.last()
	wrong := "000000"
	if wrong == code {
		wrong = "999999"
	}
	for i := 0; i < 5; i++ {
		if _, err := a.VerifyPhoneLogin(ctx, start.ChallengeToken, wrong); !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("attempt %d = %v", i, err)
		}
	}
	if _, err := a.VerifyPhoneLogin(ctx, start.ChallengeToken, code); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("attempt limit bypassed: %v", err)
	}
	var attempts int
	if err := db.QueryRow(ctx, "SELECT attempts FROM phone_login_challenges ORDER BY id DESC LIMIT 1").Scan(&attempts); err != nil || attempts != 5 {
		t.Fatalf("failed attempts were not committed: %d %v", attempts, err)
	}
	// Reissue allowed after the shared cooldown, but not within 60s.
	throttled, err := a.StartPhoneLogin(ctx, phone)
	if err != nil || len(throttled.ChallengeToken) != 43 {
		t.Fatalf("throttle leaked enrollment: %+v %v", throttled, err)
	}
	_, _, before := sms.last()
	if _, err := db.Exec(ctx, "UPDATE phone_login_challenges SET created_at=now()-interval '61 seconds'"); err != nil {
		t.Fatal(err)
	}
	next, err := a.StartPhoneLogin(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	_, nextCode, count := sms.last()
	if count != before+1 {
		t.Fatalf("second SMS was not issued: %d -> %d", before, count)
	}
	if _, err := db.Exec(ctx, "UPDATE phone_login_challenges SET created_at=now()-interval '8 minutes', expires_at=now()-interval '3 minutes' WHERE challenge_hash=$1",
		// Challenge handles are not persisted in plaintext.
		phoneLoginHash(next.ChallengeToken)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.VerifyPhoneLogin(ctx, next.ChallengeToken, nextCode); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expired OTP authenticated: %v", err)
	}
	if _, err := db.Exec(ctx, "UPDATE account_phones SET verified_at=NULL WHERE account_id=$1 AND phone=$2", accountID, phone); err != nil {
		t.Fatal(err)
	}
	unverified, err := a.StartPhoneLogin(ctx, phone)
	if err != nil || len(unverified.ChallengeToken) != 43 {
		t.Fatalf("unverified start changed public shape: %+v %v", unverified, err)
	}
	if _, err := a.VerifyPhoneLogin(ctx, unverified.ChallengeToken, nextCode); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("unverified account phone authenticated: %v", err)
	}
}

func TestPhoneLoginConcurrentSingleUse(t *testing.T) {
	_, a, sms, _, phone := phoneLoginFixture(t)
	ctx := context.Background()
	start, err := a.StartPhoneLogin(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	_, code, _ := sms.last()
	begin := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-begin
			_, err := a.VerifyPhoneLogin(ctx, start.ChallengeToken, code)
			results <- err
		}()
	}
	close(begin)
	wg.Wait()
	close(results)
	success, invalid := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, auth.ErrInvalidCredentials) {
			invalid++
		} else {
			t.Fatalf("concurrent login failed unexpectedly: %v", err)
		}
	}
	if success != 1 || invalid != 1 {
		t.Fatalf("challenge used multiple times: success=%d invalid=%d", success, invalid)
	}
}

func TestPhoneLoginHTTPUsesSharedJWTAndCookieContract(t *testing.T) {
	db, a, sms, accountID, phone := phoneLoginFixture(t)
	issuer, err := auth.NewAccessTokens(strings.Repeat("A", 32))
	if err != nil {
		t.Fatal(err)
	}
	server := api.NewServer(db, a, organization.New(db), source.NewService(db), true, "")
	server.EnableAccessTokens(issuer)
	post := func(route, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, route, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-BDSPro-Request", "1")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		return w
	}
	start := post("/auth/phone/login/start", `{"phone":"`+phone+`"}`)
	if start.Code != 202 {
		t.Fatalf("start HTTP=%d %s", start.Code, start.Body.String())
	}
	var challenge struct {
		ChallengeToken string `json:"challenge_token"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &challenge); err != nil || len(challenge.ChallengeToken) != 43 {
		t.Fatalf("challenge response: %+v %v", challenge, err)
	}
	_, code, _ := sms.last()
	verify := post("/auth/phone/login/verify", `{"challenge_token":"`+challenge.ChallengeToken+`","code":"`+code+`"}`)
	if verify.Code != 200 || len(verify.Result().Cookies()) == 0 {
		t.Fatalf("verify HTTP=%d cookies=%d %s", verify.Code, len(verify.Result().Cookies()), verify.Body.String())
	}
	var response struct {
		AccountID int64 `json:"account_id"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(verify.Body.Bytes(), &response); err != nil || response.AccountID != accountID || response.AccessToken == "" {
		t.Fatalf("same account and JWT not returned: %+v %v", response, err)
	}
	me := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	me.Header.Set("Authorization", "Bearer "+response.AccessToken)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, me)
	if w.Code != 200 {
		t.Fatalf("JWT from phone login rejected: %d %s", w.Code, w.Body.String())
	}
}

func phoneLoginHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
