package db

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	sqlc "github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// This exercises the real SQLC state transitions in a rollback-only transaction
// against the migrated disposable CI database, without a Google provider call.
func TestGoogleFlowLifecycle(t *testing.T) {
	source := os.Getenv("BDSPRO_GOOGLE_FLOW_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("requires disposable migrated BDSPRO_GOOGLE_FLOW_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	q := sqlc.New(tx)

	// Random fixture identities avoid touching unrelated rows.
	id := uuid.New()
	browser := append(append([]byte(nil), id[:]...), id[:]...)
	id = uuid.New()
	state := append(append([]byte(nil), id[:]...), id[:]...)
	id = uuid.New()
	proof := append(append([]byte(nil), id[:]...), id[:]...)
	nonce := "0123456789012345678901234567890123456789012"
	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	if err := q.CreateGoogleFlow(ctx, CreateGoogleFlowParams{
		BrowserDigest: browser, StateDigest: state, Nonce: nonce, Verifier: verifier,
	}); err != nil {
		t.Fatal(err)
	}
	checkPhase := func(want string, ttl int, consumed bool) {
		t.Helper()
		var phase string
		var remaining float64
		var cleared bool
		err := tx.QueryRow(ctx, "SELECT phase, extract(epoch FROM expires_at-clock_timestamp())::float8, state_digest IS NULL AND nonce='' AND verifier='' FROM google_flows WHERE browser_digest=$1", browser).Scan(&phase, &remaining, &cleared)
		if err != nil {
			t.Fatal(err)
		}
		if phase != want || remaining > float64(ttl) || remaining < float64(ttl-5) || cleared != consumed {
			t.Fatalf("phase=%s remaining=%f cleared=%t; want %s/%ds/%t", phase, remaining, cleared, want, ttl, consumed)
		}
	}
	checkPhase("pending", 600, false)
	if _, err := q.LockPendingGoogleFlow(ctx, LockPendingGoogleFlowParams{BrowserDigest: browser, StateDigest: bytes.Repeat([]byte{0}, 32)}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("wrong state accepted")
	}
	pending, err := q.LockPendingGoogleFlow(ctx, LockPendingGoogleFlowParams{BrowserDigest: browser, StateDigest: state})
	if err != nil || pending.Nonce != nonce || pending.Verifier != verifier {
		t.Fatal("pending binding lost", err)
	}
	if n, err := q.ClaimGoogleFlow(ctx, browser); err != nil || n != 1 {
		t.Fatal("claim failed", n, err)
	}
	checkPhase("exchanging", 120, true)
	if n, err := q.ClaimGoogleFlow(ctx, browser); err != nil || n != 0 {
		t.Fatal("claim replay accepted", n, err)
	}
	if _, err := q.LockPendingGoogleFlow(ctx, LockPendingGoogleFlowParams{BrowserDigest: browser, StateDigest: state}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("consumed state accepted")
	}
	finish := FinishGoogleFlowParams{BrowserDigest: browser, Subject: "fixture-subject"}
	if n, err := q.FinishGoogleFlow(ctx, finish); err != nil || n != 1 {
		t.Fatal("finish failed", n, err)
	}
	checkPhase("ready", 300, true)
	if n, err := q.FinishGoogleFlow(ctx, finish); err != nil || n != 0 {
		t.Fatal("finish replay accepted", n, err)
	}
	if _, err := q.LockGoogleResult(ctx, browser); err != nil {
		t.Fatal(err)
	}
	// Use a non-default TTL to prove registration uses the input policy.
	issue := IssueGoogleRegistrationProofParams{BrowserDigest: browser, ProofDigest: proof, TtlMinutes: 7}
	if n, err := q.IssueGoogleRegistrationProof(ctx, issue); err != nil || n != 1 {
		t.Fatal("registration issue failed", n, err)
	}
	checkPhase("registration", 420, true)
	if n, err := q.IssueGoogleRegistrationProof(ctx, issue); err != nil || n != 0 {
		t.Fatal("registration replay accepted", n, err)
	}
	if _, err := q.LockGoogleRegistration(ctx, LockGoogleRegistrationParams{BrowserDigest: browser, ProofDigest: state}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("wrong proof accepted")
	}
	if _, err := q.LockGoogleRegistration(ctx, LockGoogleRegistrationParams{BrowserDigest: state, ProofDigest: proof}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("wrong browser accepted")
	}
	binding := LockGoogleRegistrationParams{BrowserDigest: browser, ProofDigest: proof}
	if _, err := q.LockGoogleRegistration(ctx, binding); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "UPDATE google_flows SET expires_at=clock_timestamp()-interval '1 second' WHERE browser_digest=$1", browser); err != nil {
		t.Fatal(err)
	}
	if _, err := q.LockGoogleRegistration(ctx, binding); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("expired registration accepted")
	}
}
