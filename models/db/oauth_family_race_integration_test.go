package db

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This test deliberately holds the family lock while two independent callers
// try to rotate a live token and replay a consumed one. Neither is allowed to
// progress past a family-wide revocation that commits first.
func TestOAuthRefreshFamilySerializesRotationAndReplay(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t) // disposable PostgreSQL 18
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: clientID, OwnerAccountID: accountID, Name: "family lock test", RedirectUri: redirect,
	}); err != nil {
		t.Fatal(err)
	}
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	code := hashOAuthTest(uuid.NewString())
	if err := issueOAuthCodeForTest(t, ctx, store, clientID, accountID, redirect,
		"profile:read", challenge, code, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	family := uuid.New()
	oldAccess := hashOAuthTest(uuid.NewString())
	oldRefresh := hashOAuthTest(uuid.NewString())
	if _, err := store.RedeemOAuthAuthCode(ctx, clientID, code, redirect, challenge,
		oldAccess, oldRefresh, time.Now().Add(time.Minute), time.Now().Add(time.Hour), family); err != nil {
		t.Fatal(err)
	}
	activeAccess := hashOAuthTest(uuid.NewString())
	activeRefresh := hashOAuthTest(uuid.NewString())
	if _, err := store.RotateOAuthRefreshToken(ctx, clientID, oldRefresh,
		activeAccess, activeRefresh, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	blocker, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(context.Background())
	if err := lockOAuthFamily(ctx, blocker, family); err != nil {
		t.Fatal(err)
	}

	type callResult struct {
		op  string
		err error
	}
	started := make(chan struct{}, 2)
	completed := make(chan callResult, 2)
	candidateAccess := hashOAuthTest(uuid.NewString())
	candidateRefresh := hashOAuthTest(uuid.NewString())
	go func() {
		started <- struct{}{}
		_, err := store.RotateOAuthRefreshToken(ctx, clientID, activeRefresh,
			candidateAccess, candidateRefresh, time.Now().Add(time.Minute))
		completed <- callResult{op: "rotation", err: err}
	}()
	go func() {
		started <- struct{}{}
		_, err := store.RotateOAuthRefreshToken(ctx, clientID, oldRefresh,
			hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()), time.Now().Add(time.Minute))
		completed <- callResult{op: "replay", err: err}
	}()
	<-started
	<-started
	// Account/client serialization precedes family locking. One request
	// holds the account/client lock and waits on family; the second waits
	// on the account/client key, not on family. Verify both server-side.
	waitForOAuthFamilyWaiters(t, ctx, store, family, 1)
	waitForOAuthAuthorizationWaiters(t, ctx, store, accountID, clientID, 1)
	select {
	case result := <-completed:
		t.Fatalf("%s escaped the held family lock: %v", result.op, result.err)
	default:
	}

	// Commit revocation while both callers wait on this family lock.
	if err := revokeOAuthFamily(ctx, blocker, family); err != nil {
		t.Fatal(err)
	}
	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		select {
		case result := <-completed:
			switch result.op {
			case "rotation":
				if !errors.Is(result.err, ErrOAuthGrantInvalid) {
					t.Fatalf("rotation after revocation returned %v", result.err)
				}
			case "replay":
				if !errors.Is(result.err, ErrOAuthGrantReplay) {
					t.Fatalf("consumed refresh replay returned %v", result.err)
				}
			default:
				t.Fatalf("unexpected operation %s", result.op)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("refresh callers did not finish after releasing family lock")
		}
	}
	if _, err := store.ResolveOAuthAccessToken(ctx, candidateAccess); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("new access token survived revocation: %v", err)
	}
	if _, err := store.RotateOAuthRefreshToken(ctx, clientID, candidateRefresh,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()), time.Now().Add(time.Minute)); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("new refresh credential survived revocation: %v", err)
	}
}

// Explicit RFC 7009 refresh-token revocation must use the same family lock.
func TestOAuthExplicitRevocationWaitsForFamilyLock(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: clientID, OwnerAccountID: accountID, Name: "revoke family lock test", RedirectUri: redirect,
	}); err != nil {
		t.Fatal(err)
	}
	code := hashOAuthTest(uuid.NewString())
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	if err := issueOAuthCodeForTest(t, ctx, store, clientID, accountID, redirect,
		"profile:read", challenge, code, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	family := uuid.New()
	access := hashOAuthTest(uuid.NewString())
	refresh := hashOAuthTest(uuid.NewString())
	if _, err := store.RedeemOAuthAuthCode(ctx, clientID, code, redirect, challenge,
		access, refresh, time.Now().Add(time.Minute), time.Now().Add(time.Hour), family); err != nil {
		t.Fatal(err)
	}
	gate, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer gate.Rollback(context.Background())
	if err := lockOAuthFamily(ctx, gate, family); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{}, 1)
	completed := make(chan error, 1)
	go func() {
		started <- struct{}{}
		completed <- store.RevokeOAuthBearer(ctx, clientID, refresh)
	}()
	<-started
	waitForOAuthFamilyWaiters(t, ctx, store, family, 1)
	select {
	case err := <-completed:
		t.Fatalf("explicit revoke ignored family lock: %v", err)
	default:
	}
	if err := gate.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-completed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("explicit revoke remained blocked")
	}
	if _, err := store.ResolveOAuthAccessToken(ctx, access); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("access token survived explicit family revoke: %v", err)
	}
	if _, err := store.RotateOAuthRefreshToken(ctx, clientID, refresh,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()), time.Now().Add(time.Minute)); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("refresh token survived explicit family revoke: %v", err)
	}
}

// waitForOAuthFamilyWaiters observes server-side advisory-lock contention.
// For a 64-bit advisory lock, PostgreSQL exposes the upper/lower halves as
// classid/objid and reports objsubid=1 (PostgreSQL 18 pg_locks documentation).
// Unlike a time.Sleep assertion, this proves the tested operation reached
// its lock wait before the blocking transaction is committed.
func waitForOAuthFamilyWaiters(t *testing.T, ctx context.Context, store *SQLStore, family uuid.UUID, want int) {
	t.Helper()
	digest := sha256.Sum256(family[:])
	high := int64(binary.BigEndian.Uint32(digest[:4]))
	low := int64(binary.BigEndian.Uint32(digest[4:8]))

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		var blocked int
		err := store.pool.QueryRow(ctx, `SELECT count(*) FROM pg_catalog.pg_locks
			WHERE locktype = 'advisory' AND NOT granted
			  AND mode = 'ExclusiveLock' AND objsubid = 1
			  AND classid::bigint = $1 AND objid::bigint = $2
			  AND database = (SELECT oid FROM pg_catalog.pg_database
			                  WHERE datname = current_database())`, high, low).Scan(&blocked)
		if err != nil {
			t.Fatalf("inspect advisory lock waiters: %v", err)
		}
		if blocked >= want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("waiting for advisory lock waiters: %v", ctx.Err())
		case <-deadline.C:
			t.Fatalf("expected %d PostgreSQL advisory lock waiters, observed %d", want, blocked)
		case <-ticker.C:
		}
	}
}

// waitForOAuthAuthorizationWaiters verifies blocked PostgreSQL sessions on the
// exact bigint advisory lock for (account,client), rather than sleeping.
func waitForOAuthAuthorizationWaiters(t *testing.T, ctx context.Context, store *SQLStore, accountID int64, clientID uuid.UUID, want int) {
	t.Helper()
	key := uint64(oauthAuthorizationKey(accountID, clientID))
	high, low := int64(uint32(key>>32)), int64(uint32(key))
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked int
		err := store.pool.QueryRow(ctx, `SELECT count(*) FROM pg_catalog.pg_locks
			WHERE locktype='advisory' AND NOT granted AND mode='ExclusiveLock'
			AND objsubid=1 AND classid::bigint=$1 AND objid::bigint=$2
			AND database=(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database())`, high, low).Scan(&blocked)
		if err != nil {
			t.Fatalf("inspect account authorization lock waiters: %v", err)
		}
		if blocked >= want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("authorization lock wait cancelled: %v", ctx.Err())
		case <-deadline.C:
			t.Fatalf("expected %d authorization lock waiters, saw %d", want, blocked)
		case <-ticker.C:
		}
	}
}
