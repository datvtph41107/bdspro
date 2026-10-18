package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func coreDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("CORE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("requires CORE_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	var name string
	if err := db.QueryRow(ctx, "SELECT current_database()").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "bdspro_core_ci" {
		t.Fatalf("refusing to truncate %q (requires bdspro_core_ci)", name)
	}
	_, err = db.Exec(ctx, "TRUNCATE parties,permissions,platform_organization_reviewers RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(ctx, "INSERT INTO permissions(code) VALUES('members.read'),('members.invite'),('members.remove'),('sources.create'),('sources.read'),('sources.price_report')")
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func coreUser(t *testing.T, service *auth.Service, email string) (int64, string) {
	t.Helper()
	ctx := context.Background()
	reg, err := service.Register(ctx, email, "ComplexPassphrase123!")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyEmail(ctx, reg.VerificationToken); err != nil {
		t.Fatal(err)
	}
	login, err := service.Login(ctx, email, "ComplexPassphrase123!")
	if err != nil {
		t.Fatal(err)
	}
	if reg.AccountID != login.AccountID || login.Token == "" {
		t.Fatal("wrong login identity")
	}
	return login.AccountID, login.Token
}

func TestCoreJourneyRegisterOrganizationInviteSource(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	org := organization.New(db)
	sources := source.NewService(db)

	reviewer, _ := coreUser(t, a, "reviewer@example.test")
	an, anToken := coreUser(t, a, "an@example.test")
	lan, lanToken := coreUser(t, a, "lan@example.test")
	_, err := db.Exec(ctx, `INSERT INTO platform_organization_reviewers(account_id,authorization_reference)
 VALUES ($1,'operator-approved-test-fixture')`, reviewer)
	if err != nil {
		t.Fatal(err)
	}

	reg, err := org.SubmitRegistration(ctx, an, "ABC Realty", "VN", "AB123")
	if err != nil {
		t.Fatal(err)
	}
	// Applicant cannot claim Organization administrator rights before review.
	var organizationsBefore int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM organizations").Scan(&organizationsBefore); err != nil {
		t.Fatal(err)
	}
	if organizationsBefore != 0 {
		t.Fatalf("unexpected org before approval: %d", organizationsBefore)
	}
	if _, err := org.ApproveRegistration(ctx, an, reg, "self-approved"); !errors.Is(err, organization.ErrForbidden) {
		t.Fatalf("applicant approved self: %v", err)
	}
	abc, err := org.ApproveRegistration(ctx, reviewer, reg, "operator-verified-registry")
	if err != nil {
		t.Fatal(err)
	}
	if abc <= 0 {
		t.Fatal("no company")
	}
	if _, err := org.ApproveRegistration(ctx, reviewer, reg, "repeat"); !errors.Is(err, organization.ErrNotFound) {
		t.Fatalf("duplicate reviewer action: %v", err)
	}
	var role int64
	if err := db.QueryRow(ctx, `SELECT id FROM organization_roles
 WHERE organization_id=$1 AND name='Workspace Administrator'`, abc).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if role <= 0 {
		t.Fatal("no first admin role")
	}
	// The initial admin can invite; receiving an invitation creates no membership.
	invitation, err := org.Invite(ctx, an, abc, role, "lan@example.test")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM organization_memberships WHERE organization_id=$1", abc).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("invite granted early membership: %d", count)
	}
	membership, err := org.AcceptInvitation(ctx, lan, invitation)
	if err != nil {
		t.Fatal(err)
	}
	if membership <= 0 {
		t.Fatal("missing membership")
	}
	if _, err := org.AcceptInvitation(ctx, lan, invitation); !errors.Is(err, organization.ErrNotFound) {
		t.Fatalf("double acceptance: %v", err)
	}
	members, err := org.ListMembers(ctx, lan, abc)
	if err != nil || len(members) != 2 {
		t.Fatalf("member list: %v %#v", err, members)
	}

	// A real Session identifies the same Account seen in SQL Authorization.
	tokenAccount, err := a.AccountFromToken(ctx, lanToken)
	if err != nil || tokenAccount != lan {
		t.Fatalf("session actor: %d %v", tokenAccount, err)
	}
	s, err := sources.Capture(ctx, lan, abc, "capture-from-lan", source.CaptureInput{
		Channel: "zalo", AreaM2: "75",
		InitialPrice: &source.PriceInput{Amount: "4200000000", Currency: "VND"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.ID <= 0 {
		t.Fatal("source not inserted")
	}
	_, err = sources.ReportPrice(ctx, an, abc, s.ID, "price-after-call", source.PriceInput{Amount: "4100000000", Currency: "VND"})
	if err != nil {
		t.Fatal(err)
	}
	history, err := sources.PriceHistory(ctx, lan, abc, s.ID)
	if err != nil || len(history) != 2 {
		t.Fatalf("history: %+v %v", history, err)
	}

	// A second organization must not accept ABC's role for a different tenant.
	reg2, err := org.SubmitRegistration(ctx, reviewer, "XYZ Homes", "VN", "XYZ987")
	if err != nil {
		t.Fatal(err)
	}
	xyz, err := org.ApproveRegistration(ctx, reviewer, reg2, "verified XYZ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sources.Read(ctx, an, xyz, s.ID); !errors.Is(err, source.ErrNotFound) {
		t.Fatalf("cross-tenant read: %v", err)
	}
	if _, err := org.ListMembers(ctx, lan, xyz); !errors.Is(err, organization.ErrNotFound) {
		t.Fatalf("cross-tenant members: %v", err)
	}
	claim, err := org.SubmitClaim(ctx, lan, abc, "pretend-to-represent-abc")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := org.ApproveInitialClaim(ctx, reviewer, claim, "fake-claim"); !errors.Is(err, organization.ErrConflict) {
		t.Fatalf("existing admin was replaced: %v", err)
	}
	// End of tenancy revokes ability to read stored data.
	if _, err := db.Exec(ctx, "UPDATE organization_memberships SET ended_at=now() WHERE id=$1", membership); err != nil {
		t.Fatal(err)
	}
	if _, err := sources.Read(ctx, lan, abc, s.ID); !errors.Is(err, source.ErrNotFound) {
		t.Fatalf("revoked source: %v", err)
	}
	if err := a.Logout(ctx, anToken); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AccountFromToken(ctx, anToken); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("logout token replay: %v", err)
	}
}

// recoveryMailer simulates the external delivery boundary independently of PostgreSQL.
type recoveryMailer struct {
	attempts  int
	failNext  bool
	lastToken string
}

func (m *recoveryMailer) SendVerification(_ context.Context, _ string, token string) error {
	m.attempts++
	if m.failNext {
		m.failNext = false
		return errors.New("SMTP rejected the message")
	}
	m.lastToken = token
	return nil
}

func TestRegistrationDeliveryFailureCanRecover(t *testing.T) {
	pool := coreDB(t)
	ctx := context.Background()
	mailer := &recoveryMailer{failNext: true}
	service := auth.New(pool, mailer)
	registration, err := service.Register(ctx, "recover@example.test", "VeryLongPassword1234!")
	if !errors.Is(err, auth.ErrMailerUnavailable) || registration.AccountID <= 0 {
		t.Fatalf("expected committed account with delivery failure, got %+v, %v", registration, err)
	}
	if mailer.attempts != 1 {
		t.Fatalf("expected SMTP attempted once, got %d", mailer.attempts)
	}
	// The first verification token remains in storage despite failed delivery.
	if err := service.ResendVerification(ctx, "recover@example.test"); err != nil {
		t.Fatal(err)
	}
	if mailer.attempts != 1 {
		t.Fatal("resend must observe one-minute cooldown")
	}
	_, err = pool.Exec(ctx, `UPDATE email_verification_tokens SET created_at=now()-interval '2 minutes'
WHERE account_email_id=(SELECT id FROM account_emails WHERE account_id=$1)`, registration.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ResendVerification(ctx, "RECOVER@example.test"); err != nil {
		t.Fatal(err)
	}
	if mailer.attempts != 2 || mailer.lastToken == "" {
		t.Fatalf("resend failed, attempts=%d", mailer.attempts)
	}
	if err := service.VerifyEmail(ctx, mailer.lastToken); err != nil {
		t.Fatalf("resend token did not verify the account: %v", err)
	}
	if err := service.ResendVerification(ctx, "recover@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := service.ResendVerification(ctx, "unknown@example.test"); err != nil {
		t.Fatal(err)
	}
	if mailer.attempts != 2 {
		t.Fatal("resend leaked verified or unknown state via extra SMTP calls")
	}
}

type resendCounterMailer struct {
	calls atomic.Int64
}

func (m *resendCounterMailer) SendVerification(context.Context, string, string) error {
	m.calls.Add(1)
	return nil
}

func TestConcurrentResendDoesNotBypassCooldown(t *testing.T) {
	pool := coreDB(t)
	ctx := context.Background()
	mailer := &resendCounterMailer{}
	service := auth.New(pool, mailer)
	registration, err := service.Register(ctx, "concurrent-resend@example.test", "VeryLongPassword1234!")
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE email_verification_tokens SET created_at=now()-interval '2 minutes'
WHERE account_email_id=(SELECT id FROM account_emails WHERE account_id=$1)`, registration.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- service.ResendVerification(ctx, "concurrent-resend@example.test")
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent resend: %v", err)
		}
	}
	if got := mailer.calls.Load(); got != 2 {
		t.Fatalf("expected registration + one resend, delivered %d messages", got)
	}
	var tokens int64
	err = pool.QueryRow(ctx, `SELECT count(*) FROM email_verification_tokens
WHERE account_email_id=(SELECT id FROM account_emails WHERE account_id=$1)`, registration.AccountID).Scan(&tokens)
	if err != nil || tokens != 2 {
		t.Fatalf("concurrent resend issued %d tokens: %v", tokens, err)
	}
}

func TestResendDailyTokenCap(t *testing.T) {
	pool := coreDB(t)
	ctx := context.Background()
	mailer := &resendCounterMailer{}
	service := auth.New(pool, mailer)
	registration, err := service.Register(ctx, "daily-resend@example.test", "VeryLongPassword1234!")
	if err != nil {
		t.Fatal(err)
	}
	var emailID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM account_emails WHERE account_id=$1", registration.AccountID).Scan(&emailID); err != nil {
		t.Fatal(err)
	}
	// Registration consumes one of six token issuances in the last 24 hours.
	for i := 0; i < 5; i++ {
		_, err := pool.Exec(ctx, `INSERT INTO email_verification_tokens(account_email_id, token_hash, created_at, expires_at)
VALUES($1, $2, now()-interval '2 minutes', now()+interval '30 minutes')`, emailID, fmt.Sprintf("daily-fixture-%d", i))
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := service.ResendVerification(ctx, "daily-resend@example.test"); err != nil {
		t.Fatal(err)
	}
	if got := mailer.calls.Load(); got != 1 {
		t.Fatalf("24-hour quota ignored; SMTP sent %d emails", got)
	}
}

func TestC01DeferredSubtypeAndRegistrationRollback(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "INSERT INTO parties(kind) VALUES('PERSON')"); err != nil {
		t.Fatal(err)
	}
	err = tx.Commit(ctx)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("expected C01 at COMMIT, got %v", err)
	}
	authService := auth.New(db, nil)
	_, err = authService.Register(ctx, "duplicate@example.test", "ValidLongPassword333!")
	if err != nil {
		t.Fatal(err)
	}
	var before int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&before); err != nil {
		t.Fatal(err)
	}
	_, err = authService.Register(ctx, "DUPLICATE@example.test", "ValidLongPassword333!")
	if !errors.Is(err, auth.ErrEmailUsed) {
		t.Fatalf("expected duplicate email: %v", err)
	}
	var after int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("registration left orphan party: before=%d after=%d", before, after)
	}
}

func TestAuthCookieLifecycle(t *testing.T) {
	db := coreDB(t)
	service := auth.New(db, nil)
	server := api.NewServer(db, service, organization.New(db), source.NewService(db), true, "")
	act := func(method, path string, body []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		return w
	}
	payload := []byte(`{"email":"minh@example.test","password":"VeryLongPassword1234!"}`)
	registered := act("POST", "/auth/register", payload, nil)
	if registered.Code != 201 {
		t.Fatalf("register HTTP: %d %s", registered.Code, registered.Body.String())
	}
	var reg struct {
		VerificationToken string `json:"verification_token"`
	}
	if err := json.Unmarshal(registered.Body.Bytes(), &reg); err != nil {
		t.Fatal(err)
	}
	v, _ := json.Marshal(map[string]string{"token": reg.VerificationToken})
	if r := act("POST", "/auth/verify-email", v, nil); r.Code != 204 {
		t.Fatalf("verify HTTP: %d %s", r.Code, r.Body.String())
	}
	logged := act("POST", "/auth/login", payload, nil)
	if logged.Code != 200 {
		t.Fatalf("login HTTP: %d %s", logged.Code, logged.Body.String())
	}
	cookies := logged.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie")
	}
	me := act("GET", "/auth/me", nil, cookies[0])
	if me.Code != 200 {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}
	logout := act("POST", "/auth/logout", nil, cookies[0])
	if logout.Code != 204 {
		t.Fatalf("logout %d", logout.Code)
	}
	me = act("GET", "/auth/me", nil, cookies[0])
	if me.Code != 401 {
		t.Fatalf("replayed cookie after logout: %d", me.Code)
	}
}

func TestInvitationDoesNotCrossTenant(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	a := auth.New(db, nil)
	org := organization.New(db)
	reviewer, _ := coreUser(t, a, "reviewer-scope@example.test")
	actor, _ := coreUser(t, a, "actor-scope@example.test")
	_, err := db.Exec(ctx, "INSERT INTO platform_organization_reviewers(account_id,authorization_reference) VALUES($1,'reviewer-fixture')", reviewer)
	if err != nil {
		t.Fatal(err)
	}
	regA, err := org.SubmitRegistration(ctx, actor, "Scope-A", "VN", "SCOPEA")
	if err != nil {
		t.Fatal(err)
	}
	abc, err := org.ApproveRegistration(ctx, reviewer, regA, "proof")
	if err != nil {
		t.Fatal(err)
	}
	regB, err := org.SubmitRegistration(ctx, reviewer, "Scope-B", "VN", "SCOPEB")
	if err != nil {
		t.Fatal(err)
	}
	xyz, err := org.ApproveRegistration(ctx, reviewer, regB, "proof")
	if err != nil {
		t.Fatal(err)
	}
	var roleB int64
	if err := db.QueryRow(ctx, "SELECT id FROM organization_roles WHERE organization_id=$1", xyz).Scan(&roleB); err != nil {
		t.Fatal(err)
	}
	if _, err := org.Invite(ctx, actor, abc, roleB, "stranger@example.test"); !errors.Is(err, organization.ErrForbidden) {
		t.Fatalf("cross organization role grant: %v", err)
	}
	// Raw insert is also protected by composite FK.
	var memberA int64
	if err := db.QueryRow(ctx, "SELECT id FROM organization_memberships WHERE organization_id=$1", abc).Scan(&memberA); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(ctx, "INSERT INTO organization_role_assignments(organization_id,membership_id,role_id) VALUES($1,$2,$3)", abc, memberA, roleB)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("expected role-scope FK 23503: %v", err)
	}
}

// TestConcurrentOrganizationRegistrationApproval demonstrates row-level locking
// on a real BDSPro approval, rather than a fictitious money transfer.
func TestConcurrentOrganizationRegistrationApproval(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	accounts := auth.New(db, nil)
	organizations := organization.New(db)

	reviewerID, _ := coreUser(t, accounts, "concurrent-reviewer@example.test")
	applicantID, _ := coreUser(t, accounts, "concurrent-applicant@example.test")

	_, err := db.Exec(ctx, `INSERT INTO platform_organization_reviewers
		(account_id, authorization_reference)
		VALUES ($1, 'CI-only-authorization')`, reviewerID)
	if err != nil {
		t.Fatal(err)
	}

	requestID, err := organizations.SubmitRegistration(
		ctx, applicantID, "Concurrent Realty", "VN", "CONCURRENT-001",
	)
	if err != nil {
		t.Fatal(err)
	}

	type approval struct {
		organizationID int64
		err            error
	}
	results := make(chan approval, 2)
	for i := 0; i < 2; i++ {
		go func() {
			id, err := organizations.ApproveRegistration(
				ctx, reviewerID, requestID, "CI-reviewed",
			)
			results <- approval{organizationID: id, err: err}
		}()
	}

	approved := 0
	notFound := 0
	for i := 0; i < 2; i++ {
		result := <-results
		switch {
		case result.err == nil:
			if result.organizationID <= 0 {
				t.Fatal("approval did not return an organization")
			}
			approved++
		case errors.Is(result.err, organization.ErrNotFound):
			notFound++
		default:
			t.Fatalf("unexpected concurrent approval error: %v", result.err)
		}
	}
	if approved != 1 || notFound != 1 {
		t.Fatalf("expected one approval and one rejected duplicate: approved=%d not_found=%d", approved, notFound)
	}

	var count int
	if err := db.QueryRow(ctx,
		"SELECT count(*) FROM organizations WHERE name='Concurrent Realty'",
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate organization from concurrent approval: %d", count)
	}
}

func TestGinOrganizationAndSourceHTTP(t *testing.T) {
	pool := coreDB(t)
	ctx := context.Background()
	a := auth.New(pool, nil)
	org := organization.New(pool)
	sources := source.NewService(pool)
	server := api.NewServer(pool, a, org, sources, true, "https://app.example.test")

	reviewerID, reviewerToken := coreUser(t, a, "gin-reviewer@example.test")
	_, anToken := coreUser(t, a, "gin-an@example.test")
	_, lanToken := coreUser(t, a, "gin-lan@example.test")
	_, err := pool.Exec(ctx,
		"INSERT INTO platform_organization_reviewers(account_id,authorization_reference) VALUES($1,'reviewed-fixture')",
		reviewerID)
	if err != nil {
		t.Fatal(err)
	}

	call := func(method, path, token, key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "bdspro_session", Value: token})
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, r)
		return w
	}
	var readID = func(t *testing.T, w *httptest.ResponseRecorder, field string) int64 {
		t.Helper()
		var body map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		var id int64
		if err := json.Unmarshal(body[field], &id); err != nil || id <= 0 {
			t.Fatalf("missing %s: %s (%v)", field, w.Body.String(), err)
		}
		return id
	}

	registrationBody := `{"name":"Gin Realty","jurisdiction":"VN","registration_code":"GIN-001"}`
	if w := call("POST", "/organizations/registrations", "", "", registrationBody); w.Code != 401 {
		t.Fatalf("anonymous registration: %d", w.Code)
	}
	registration := call("POST", "/organizations/registrations", anToken, "", registrationBody)
	if registration.Code != 201 {
		t.Fatalf("register organization: %d %s", registration.Code, registration.Body.String())
	}
	requestID := readID(t, registration, "request_id")

	approveURL := fmt.Sprintf("/platform/organizations/registrations/%d/approve", requestID)
	if w := call("POST", approveURL, anToken, "", `{"review_reference":"test"}`); w.Code != 404 {
		t.Fatalf("non-reviewer approved request: %d %s", w.Code, w.Body.String())
	}
	approved := call("POST", approveURL, reviewerToken, "", `{"review_reference":"reviewed"}`)
	if approved.Code != 200 {
		t.Fatalf("approve organization: %d %s", approved.Code, approved.Body.String())
	}
	orgID := readID(t, approved, "organization_id")

	// The second registration attempt to represent the organization must
	// not silently steal the incumbent administrator.
	claimURL := fmt.Sprintf("/organizations/%d/claims", orgID)
	claim := call("POST", claimURL, lanToken, "", `{"evidence_reference":"unverified-claim"}`)
	if claim.Code != 201 {
		t.Fatalf("claim request: %d %s", claim.Code, claim.Body.String())
	}
	claimID := readID(t, claim, "request_id")
	approveClaim := fmt.Sprintf("/platform/organizations/claims/%d/approve", claimID)
	if w := call("POST", approveClaim, reviewerToken, "", `{"review_reference":"deny-claim"}`); w.Code != 409 {
		t.Fatalf("incumbent admin displaced: %d %s", w.Code, w.Body.String())
	}

	var roleID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM organization_roles WHERE organization_id=$1", orgID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	inviteURL := fmt.Sprintf("/organizations/%d/invitations", orgID)
	inviteBody := fmt.Sprintf(`{"email":"gin-lan@example.test","role_id":%d}`, roleID)
	invite := call("POST", inviteURL, anToken, "", inviteBody)
	if invite.Code != 201 {
		t.Fatalf("invite: %d %s", invite.Code, invite.Body.String())
	}
	invitationID := readID(t, invite, "invitation_id")
	acceptURL := fmt.Sprintf("/organizations/invitations/%d/accept", invitationID)
	if w := call("POST", acceptURL, lanToken, "", ""); w.Code != 200 {
		t.Fatalf("accept invitation: %d %s", w.Code, w.Body.String())
	}
	membersURL := fmt.Sprintf("/organizations/%d/members", orgID)
	if w := call("GET", membersURL, lanToken, "", ""); w.Code != 200 {
		t.Fatalf("list members: %d %s", w.Code, w.Body.String())
	}

	sourceURL := fmt.Sprintf("/organizations/%d/sources", orgID)
	captureBody := `{"channel":"zalo","label":"House","area_m2":"75","initial_price":{"amount":"4200000000","currency":"VND"}}`
	first := call("POST", sourceURL, lanToken, "gin-capture-1", captureBody)
	if first.Code != 201 {
		t.Fatalf("capture: %d %s", first.Code, first.Body.String())
	}
	sourceID := readID(t, first, "id")
	if w := call("POST", sourceURL, lanToken, "gin-capture-1", captureBody); w.Code != 200 {
		t.Fatalf("idempotent replay: %d %s", w.Code, w.Body.String())
	}
	if w := call("POST", sourceURL, lanToken, "gin-capture-1", `{"channel":"phone"}`); w.Code != 409 {
		t.Fatalf("idempotency mismatch: %d %s", w.Code, w.Body.String())
	}

	getSourceURL := fmt.Sprintf("%s/%d", sourceURL, sourceID)
	if w := call("GET", getSourceURL, lanToken, "", ""); w.Code != 200 {
		t.Fatalf("read source: %d %s", w.Code, w.Body.String())
	}
	if w := call("GET", getSourceURL, reviewerToken, "", ""); w.Code != 404 {
		t.Fatalf("cross-tenant source read: %d %s", w.Code, w.Body.String())
	}
	priceURL := getSourceURL + "/price-reports"
	if w := call("POST", priceURL, lanToken, "gin-price-1", `{"amount":"4100000000","currency":"VND"}`); w.Code != 201 {
		t.Fatalf("report price: %d %s", w.Code, w.Body.String())
	}
	if w := call("GET", priceURL, lanToken, "", ""); w.Code != 200 {
		t.Fatalf("price history: %d %s", w.Code, w.Body.String())
	}
	if w := call("POST", priceURL, lanToken, "gin-price-2", `{"amount":"100","currency":"VND"}{"extra":true}`); w.Code != 400 {
		t.Fatalf("accepted trailing JSON: %d %s", w.Code, w.Body.String())
	}
	if w := call("GET", "/organizations/0/sources/1", lanToken, "", ""); w.Code != 400 {
		t.Fatalf("invalid route ID: %d %s", w.Code, w.Body.String())
	}

	crossSite := httptest.NewRequest(http.MethodPost, sourceURL, bytes.NewBufferString(captureBody))
	crossSite.AddCookie(&http.Cookie{Name: "bdspro_session", Value: lanToken})
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	denied := httptest.NewRecorder()
	server.Handler().ServeHTTP(denied, crossSite)
	if denied.Code != 403 {
		t.Fatalf("cross-site write: %d %s", denied.Code, denied.Body.String())
	}
}

func TestRegistryIdentityConflictDoesNotLeaveOrphanParty(t *testing.T) {
	pool := coreDB(t)
	ctx := context.Background()
	accounts := auth.New(pool, nil)
	organizations := organization.New(pool)

	reviewerID, _ := coreUser(t, accounts, "sqlstate-reviewer@example.test")
	firstID, _ := coreUser(t, accounts, "sqlstate-first@example.test")
	secondID, _ := coreUser(t, accounts, "sqlstate-second@example.test")
	_, err := pool.Exec(ctx, `INSERT INTO platform_organization_reviewers(account_id,authorization_reference)
VALUES ($1,'sqlstate-fixture')`, reviewerID)
	if err != nil {
		t.Fatal(err)
	}
	first, err := organizations.SubmitRegistration(ctx, firstID, "First Realty", "VN", "SAME-REG-001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := organizations.ApproveRegistration(ctx, reviewerID, first, "approved"); err != nil {
		t.Fatal(err)
	}
	var before int64
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&before); err != nil {
		t.Fatal(err)
	}
	second, err := organizations.SubmitRegistration(ctx, secondID, "Second Realty", "VN", "SAME-REG-001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := organizations.ApproveRegistration(ctx, reviewerID, second, "duplicate"); !errors.Is(err, organization.ErrConflict) {
		t.Fatalf("expected registry unique violation to map to business conflict, got %v", err)
	}
	var after int64
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("duplicate registration committed an orphan Party: before=%d after=%d", before, after)
	}
}

func TestClaimApprovalDatabaseTimeoutIsNotBusinessConflict(t *testing.T) {
	pool := coreDB(t)
	ctx := context.Background()
	accounts := auth.New(pool, nil)
	organizations := organization.New(pool)

	reviewerID, _ := coreUser(t, accounts, "sqlstate-claim-reviewer@example.test")
	ownerID, _ := coreUser(t, accounts, "sqlstate-claim-owner@example.test")
	claimantID, _ := coreUser(t, accounts, "sqlstate-claimant@example.test")
	_, err := pool.Exec(ctx, `INSERT INTO platform_organization_reviewers(account_id,authorization_reference)
VALUES ($1,'timeout-fixture')`, reviewerID)
	if err != nil {
		t.Fatal(err)
	}
	regID, err := organizations.SubmitRegistration(ctx, ownerID, "Timeout Realty", "VN", "TIMEOUT-001")
	if err != nil {
		t.Fatal(err)
	}
	orgID, err := organizations.ApproveRegistration(ctx, reviewerID, regID, "reviewed")
	if err != nil {
		t.Fatal(err)
	}
	claimID, err := organizations.SubmitClaim(ctx, claimantID, orgID, "claim reference")
	if err != nil {
		t.Fatal(err)
	}
	// Hold the row needed by ApproveInitialClaim, forcing the service's
	// organization lock to exceed its request deadline.
	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(context.Background())
	var locked int64
	if err := blocker.QueryRow(ctx,
		"SELECT party_id FROM organizations WHERE party_id=$1 FOR UPDATE",
		orgID).Scan(&locked); err != nil {
		t.Fatal(err)
	}
	deadline, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	_, err = organizations.ApproveInitialClaim(deadline, reviewerID, claimID, "timeout")
	if err == nil || errors.Is(err, organization.ErrConflict) {
		t.Fatalf("backend timeout must not be translated into business conflict: %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected request deadline to survive transaction cleanup: %v", err)
	}
}
