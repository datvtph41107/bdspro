package db

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newOAuthSecondaryAccount(t *testing.T, ctx context.Context, store *SQLStore) int64 {
	t.Helper()
	var id, person int64
	if err := store.pool.QueryRow(ctx, `INSERT INTO parties DEFAULT VALUES RETURNING id`).Scan(&person); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO persons(party_id) VALUES($1)`, person); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(ctx, `INSERT INTO accounts(person_id,email)
		VALUES($1,$2) RETURNING id`, person, "oauth-authorization-"+uuid.NewString()+"@example.test").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestOAuthAuthorizationsAreUserScopedAndRevoked(t *testing.T) {
	ctx, store, owner := oauthTestStore(t)
	user := newOAuthSecondaryAccount(t, ctx, store)
	client := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{ID: client, OwnerAccountID: owner, Name: "external app", RedirectUri: redirect}); err != nil {
		t.Fatal(err)
	}
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	code := hashOAuthTest(uuid.NewString())
	if err := store.CreateOAuthAuthCode(ctx, client, user, redirect, "profile:read", challenge, code, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	access, refresh := hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString())
	if _, err := store.RedeemOAuthAuthCode(ctx, client, code, redirect, challenge,
		access, refresh, time.Now().Add(time.Minute), time.Now().Add(time.Hour), uuid.New()); err != nil {
		t.Fatal(err)
	}
	pending := hashOAuthTest(uuid.NewString())
	if err := store.CreateOAuthAuthCode(ctx, client, user, redirect, "profile:read", challenge, pending, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	mine, err := store.ListOAuthAuthorizations(ctx, user)
	if err != nil || len(mine) != 1 || mine[0].ClientID != client || mine[0].Name != "external app" {
		t.Fatalf("authorized user's list = %+v, %v", mine, err)
	}
	owned, err := store.ListOAuthAuthorizations(ctx, owner)
	if err != nil || len(owned) != 0 {
		t.Fatalf("owner leaked authorized users: %+v, %v", owned, err)
	}
	if err := store.RevokeOAuthAuthorization(ctx, owner, client); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveOAuthAccessToken(ctx, access); err != nil {
		t.Fatalf("owner revoked another user's credential: %v", err)
	}
	if err := store.RevokeOAuthAuthorization(ctx, user, client); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeOAuthAuthorization(ctx, user, client); err != nil {
		t.Fatalf("revoke not idempotent: %v", err)
	}
	if _, err := store.ResolveOAuthAccessToken(ctx, access); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("access survived revoke: %v", err)
	}
	if _, err := store.RotateOAuthRefreshToken(ctx, client, refresh,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()), time.Now().Add(time.Minute)); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("refresh survived revoke: %v", err)
	}
	if _, err := store.RedeemOAuthAuthCode(ctx, client, pending, redirect, challenge,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()), time.Now().Add(time.Minute), time.Now().Add(time.Hour), uuid.New()); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("pending code survived revoke: %v", err)
	}
	mine, err = store.ListOAuthAuthorizations(ctx, user)
	if err != nil || len(mine) != 0 {
		t.Fatalf("revoked authorization still listed: %+v, %v", mine, err)
	}
}

// Both possible serial orders are safe: refresh first -> revoke destroys
// the newly issued token; revoke first -> refresh is rejected.
func TestOAuthRevokeVsRefreshNeverLeavesLiveToken(t *testing.T) {
	ctx, store, owner := oauthTestStore(t)
	user := newOAuthSecondaryAccount(t, ctx, store)
	client := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{ID: client, OwnerAccountID: owner, Name: "race app", RedirectUri: redirect}); err != nil {
		t.Fatal(err)
	}
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	code := hashOAuthTest(uuid.NewString())
	if err := store.CreateOAuthAuthCode(ctx, client, user, redirect, "profile:read", challenge, code, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	access, refresh := hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString())
	if _, err := store.RedeemOAuthAuthCode(ctx, client, code, redirect, challenge,
		access, refresh, time.Now().Add(time.Minute), time.Now().Add(time.Hour), uuid.New()); err != nil {
		t.Fatal(err)
	}
	newAccess, newRefresh := hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString())
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	var refreshErr, revokeErr error
	go func() {
		defer wg.Done()
		<-start
		_, refreshErr = store.RotateOAuthRefreshToken(ctx, client, refresh, newAccess, newRefresh, time.Now().Add(time.Minute))
	}()
	go func() { defer wg.Done(); <-start; revokeErr = store.RevokeOAuthAuthorization(ctx, user, client) }()
	close(start)
	wg.Wait()
	if revokeErr != nil {
		t.Fatalf("revoke failed: %v", revokeErr)
	}
	if refreshErr != nil && !errors.Is(refreshErr, ErrOAuthGrantInvalid) {
		t.Fatalf("refresh unexpected failure: %v", refreshErr)
	}
	for _, token := range [][]byte{access, newAccess} {
		if _, err := store.ResolveOAuthAccessToken(ctx, token); !errors.Is(err, ErrOAuthGrantInvalid) {
			t.Fatalf("access survived concurrent revoke: %v", err)
		}
	}
	for _, token := range [][]byte{refresh, newRefresh} {
		if _, err := store.RotateOAuthRefreshToken(ctx, client, token,
			hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()), time.Now().Add(time.Minute)); !errors.Is(err, ErrOAuthGrantInvalid) && !errors.Is(err, ErrOAuthGrantReplay) {
			t.Fatalf("refresh survived concurrent revoke: %v", err)
		}
	}
}
