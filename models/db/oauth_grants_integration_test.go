package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test fixtures create an actual authority session rather than bypassing the
// model transaction's session-liveness requirement.
func createOAuthTestSession(t *testing.T, ctx context.Context, store *SQLStore, accountID int64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := store.CreateSession(ctx, CreateSessionParams{
		ID: id, AccountID: accountID, RefreshToken: "oauth-model-"+uuid.NewString(),
		UserAgent: "oauth-model-test", ClientIp: "127.0.0.1",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil { t.Fatal(err) }
	return id
}

// Existing model integration tests must issue codes via the same atomic
// consent operation used by the authorization service, never bypass it.
func issueOAuthCodeForTest(t *testing.T, ctx context.Context, store *SQLStore, id uuid.UUID,
	accountID int64, redirect, scope, challenge string, hash []byte, expires time.Time) error {
	t.Helper()
	current, err := store.GetOAuthConsentVersion(ctx, accountID, id)
	if err != nil {
		return err
	}
	sessionID := createOAuthTestSession(t, ctx, store, accountID)
	return store.ApproveOAuthAuthCode(ctx, id, accountID, redirect, scope, challenge, hash, expires, current, sessionID)
}

func TestOAuthStaleConsentCannotAuthorizeAfterRevoke(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	sessionID := createOAuthTestSession(t, ctx, store, accountID)
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{ID: clientID, OwnerAccountID: accountID, Name: "consent version test", RedirectUri: redirect}); err != nil {
		t.Fatal(err)
	}
	stale, err := store.GetOAuthConsentVersion(ctx, accountID, clientID)
	if err != nil || stale != 0 {
		t.Fatalf("initial consent version %d, %v", stale, err)
	}
	if err := store.RevokeOAuthAuthorization(ctx, accountID, clientID); err != nil {
		t.Fatal(err)
	}
	current, err := store.GetOAuthConsentVersion(ctx, accountID, clientID)
	if err != nil || current <= stale {
		t.Fatalf("revoke did not persist consent version: %d, %v", current, err)
	}
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	code := hashOAuthTest(uuid.NewString())
	if err := store.ApproveOAuthAuthCode(ctx, clientID, accountID, redirect, "profile:read", challenge, code, time.Now().Add(time.Minute), stale, sessionID); !errors.Is(err, ErrOAuthConsentStale) {
		t.Fatalf("stale approval accepted: %v", err)
	}
	if err := store.ApproveOAuthAuthCode(ctx, clientID, accountID, redirect, "profile:read", challenge, code, time.Now().Add(time.Minute), current, sessionID); err != nil {
		t.Fatal(err)
	}
	clients, err := store.ListOAuthAuthorizations(ctx, accountID)
	if err != nil || len(clients) != 1 || clients[0].ClientID != clientID {
		t.Fatalf("fresh approval not listed: %+v %v", clients, err)
	}
	if err := store.RevokeOAuthAuthorization(ctx, accountID, clientID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RedeemOAuthAuthCode(ctx, clientID, code, redirect, challenge,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()),
		time.Now().Add(time.Minute), time.Now().Add(time.Hour), uuid.New()); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("pending code survived revoke: %v", err)
	}
}

// Failure at the LAST INSERT must roll back the already-updated consent epoch.
func TestOAuthConsentAndCodeRollbackTogether(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	sessionID := createOAuthTestSession(t, ctx, store, accountID)
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{ID: clientID, OwnerAccountID: accountID, Name: "rollback consent", RedirectUri: redirect}); err != nil {
		t.Fatal(err)
	}
	code := hashOAuthTest(uuid.NewString())
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	expiry := time.Now().Add(time.Minute)
	if err := store.ApproveOAuthAuthCode(ctx, clientID, accountID, redirect, "profile:read", challenge, code, expiry, 0, sessionID); err != nil {
		t.Fatal(err)
	}
	version, err := store.GetOAuthConsentVersion(ctx, accountID, clientID)
	if err != nil || version != 1 {
		t.Fatalf("initial approval version=%d error=%v", version, err)
	}
	// Same code_hash violates the PK after the grant row has been updated.
	if err := store.ApproveOAuthAuthCode(ctx, clientID, accountID, redirect, "profile:read", challenge, code, expiry, version, sessionID); err == nil {
		t.Fatal("duplicate code insert was accepted")
	}
	after, err := store.GetOAuthConsentVersion(ctx, accountID, clientID)
	if err != nil || after != version {
		t.Fatalf("failed code insert advanced consent: %d -> %d (%v)", version, after, err)
	}
}
