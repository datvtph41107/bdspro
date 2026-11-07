package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func assertOAuthRowCount(t *testing.T, ctx context.Context, store *SQLStore, query string, hash []byte, want int) {
	t.Helper()
	var count int
	err := store.pool.QueryRow(ctx, query, hash).Scan(&count)
	if err != nil || count != want {
		t.Fatalf("OAuth row count=%d, want=%d, error=%v", count, want, err)
	}
}

func TestOAuthCleanupPreservesLiveRefreshAndReclaimsExpiredRows(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t) // disposable PG18 only
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: clientID, OwnerAccountID: accountID,
		Name: "retention test", RedirectUri: redirect,
	}); err != nil {
		t.Fatal(err)
	}

	// A historical cutoff avoids deleting other tests' current credentials.
	cutoff := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	old := cutoff.Add(-time.Hour)
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"

	unusedCode := hashOAuthTest(uuid.NewString())
	if err := store.CreateOAuthAuthCode(ctx, clientID, accountID,
		redirect, "profile:read", challenge, unusedCode, old); err != nil {
		t.Fatal(err)
	}

	newGrant := func(refreshExpiry time.Time) (access, refresh []byte) {
		t.Helper()
		code := hashOAuthTest(uuid.NewString())
		if err := store.CreateOAuthAuthCode(ctx, clientID, accountID,
			redirect, "profile:read", challenge, code, time.Now().Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		access = hashOAuthTest(uuid.NewString())
		refresh = hashOAuthTest(uuid.NewString())
		if _, err := store.RedeemOAuthAuthCode(ctx, clientID, code, redirect, challenge,
			access, refresh, old, refreshExpiry, uuid.New()); err != nil {
			t.Fatal(err)
		}
		return access, refresh
	}
	liveAccess, liveRefresh := newGrant(time.Now().Add(time.Hour))
	expiredAccess, expiredRefresh := newGrant(old)

	counts, err := store.CleanupExpiredOAuth(ctx, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if counts.AuthCodes < 1 || counts.RefreshTokens < 1 || counts.AccessTokens < 1 {
		t.Fatalf("expected all three expiry classes to be swept: %+v", counts)
	}
	assertOAuthRowCount(t, ctx, store, "SELECT count(*) FROM oauth_auth_codes WHERE code_hash=$1", unusedCode, 0)
	assertOAuthRowCount(t, ctx, store, "SELECT count(*) FROM oauth_refresh_tokens WHERE token_hash=$1", expiredRefresh, 0)
	assertOAuthRowCount(t, ctx, store, "SELECT count(*) FROM oauth_access_tokens WHERE token_hash=$1", expiredAccess, 0)
	assertOAuthRowCount(t, ctx, store, "SELECT count(*) FROM oauth_refresh_tokens WHERE token_hash=$1", liveRefresh, 1)
	assertOAuthRowCount(t, ctx, store, "SELECT count(*) FROM oauth_access_tokens WHERE token_hash=$1", liveAccess, 1)

	// Deleting an expired access token without this guard would cascade away
	// its live refresh token and break rotation.
	if _, err := store.RotateOAuthRefreshToken(ctx, clientID, liveRefresh,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()),
		time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("live refresh token stopped working after cleanup: %v", err)
	}
}

func TestOAuthCleanupRejectsZeroCutoff(t *testing.T) {
	var store SQLStore
	if _, err := store.CleanupExpiredOAuth(context.Background(), time.Time{}); err == nil {
		t.Fatal("zero cutoff accepted")
	}
}
