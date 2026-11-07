package db

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Unique-key collisions are deterministic PostgreSQL faults injected after
// a grant has been locked and (for code/refresh) marked consumed. No triggers
// or transaction test hooks are needed.
func TestOAuthGrantWritesRollbackOnTokenInsertFailure(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: clientID, OwnerAccountID: accountID,
		Name: "atomicity test", RedirectUri: redirect,
	}); err != nil {
		t.Fatal(err)
	}
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	newCode := func() []byte {
		t.Helper()
		code := hashOAuthTest(uuid.NewString())
		if err := issueOAuthCodeForTest(t, ctx, store, clientID, accountID, redirect,
			"profile:read", challenge, code, time.Now().Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		return code
	}
	access := hashOAuthTest(uuid.NewString())
	refresh := hashOAuthTest(uuid.NewString())
	if _, err := store.RedeemOAuthAuthCode(ctx, clientID, newCode(), redirect, challenge,
		access, refresh, time.Now().Add(time.Minute), time.Now().Add(time.Hour), uuid.New()); err != nil {
		t.Fatal(err)
	}

	// An access-token PK collision occurs after the code has been UPDATEd.
	secondCode := newCode()
	_, err := store.RedeemOAuthAuthCode(ctx, clientID, secondCode, redirect, challenge,
		access, hashOAuthTest(uuid.NewString()), time.Now().Add(time.Minute),
		time.Now().Add(time.Hour), uuid.New())
	assertOAuthUniqueViolation(t, err)

	// If the failed operation consumed the code outside its transaction,
	// a retry with a unique access token would return invalid_grant.
	if _, err := store.RedeemOAuthAuthCode(ctx, clientID, secondCode, redirect, challenge,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()),
		time.Now().Add(time.Minute), time.Now().Add(time.Hour), uuid.New()); err != nil {
		t.Fatalf("failed grant insert consumed the code: %v", err)
	}

	// Another collision happens AFTER writing a new access token, when the
	// refresh-token INSERT hits the unique PK constraint.
	transientAccess := hashOAuthTest(uuid.NewString())
	_, err = store.RotateOAuthRefreshToken(ctx, clientID, refresh,
		transientAccess, refresh, time.Now().Add(time.Minute))
	assertOAuthUniqueViolation(t, err)
	assertOAuthRowCount(t, ctx, store,
		"SELECT count(*) FROM oauth_access_tokens WHERE token_hash=$1", transientAccess, 0)

	// The old refresh token must remain usable after the failed rotation.
	if _, err := store.RotateOAuthRefreshToken(ctx, clientID, refresh,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()),
		time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("failed refresh insert consumed the original refresh token: %v", err)
	}
}

func assertOAuthUniqueViolation(t *testing.T, err error) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("want a PostgreSQL unique-key constraint fault, got %v", err)
	}
}
