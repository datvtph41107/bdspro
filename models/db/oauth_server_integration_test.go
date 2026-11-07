package db

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func oauthTestStore(t *testing.T) (context.Context, *SQLStore, int64) {
	t.Helper()
	raw := os.Getenv("BDS_TEST_DB_URL")
	if raw == "" {
		t.Skip("requires disposable PostgreSQL 18 database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing OAuth integration outside disposable mode")
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatalf("refusing writes outside bdspro_test: %s", cfg.ConnConfig.Database)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	var personID, accountID int64
	if err := pool.QueryRow(ctx, "INSERT INTO parties DEFAULT VALUES RETURNING id").Scan(&personID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO persons(party_id) VALUES($1)", personID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO accounts(person_id,email) VALUES($1,$2) RETURNING id", personID, "oauth-integration-"+uuid.NewString()+"@example.test").Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	return ctx, NewStore(pool), accountID
}

func hashOAuthTest(s string) []byte {
	d := sha256.Sum256([]byte(s))
	return d[:]
}

func TestOAuthCodeSingleUseAndRefreshReplay(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	id := uuid.New()
	redirect := "https://client.example.test/callback"
	client, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: id, OwnerAccountID: accountID, Name: "test client", RedirectUri: redirect,
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.ID != id {
		t.Fatal("persisted OAuth client ID differs")
	}
	code := hashOAuthTest(uuid.NewString())
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	if len(challenge) != 43 {
		t.Fatal("test challenge must be 43 chars")
	}
	if err := issueOAuthCodeForTest(t, ctx, store, id, accountID, redirect,
		"profile:read", challenge, code, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	access := hashOAuthTest(uuid.NewString())
	refresh := hashOAuthTest(uuid.NewString())
	family := uuid.New()
	expires := time.Now().Add(time.Minute)
	refreshExpires := time.Now().Add(time.Hour)
	if _, err := store.RedeemOAuthAuthCode(ctx, id, code, redirect,
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", access, refresh,
		expires, refreshExpires, family); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("wrong verifier accepted: %v", err)
	}
	grant, err := store.RedeemOAuthAuthCode(ctx, id, code, redirect, challenge,
		access, refresh, expires, refreshExpires, family)
	if err != nil || grant.AccountID != accountID || grant.Scope != "profile:read" {
		t.Fatalf("valid code exchange failed: grant=%+v error=%v", grant, err)
	}
	if _, err := store.RedeemOAuthAuthCode(ctx, id, code, redirect, challenge,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()),
		expires, refreshExpires, uuid.New()); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("replayed code accepted: %v", err)
	}
	if accessGrant, err := store.ResolveOAuthAccessToken(ctx, access); err != nil ||
		accessGrant.AccountID != accountID || accessGrant.Scope != "profile:read" {
		t.Fatalf("new access grant cannot be resolved: %+v %v", accessGrant, err)
	}
	nextAccess := hashOAuthTest(uuid.NewString())
	nextRefresh := hashOAuthTest(uuid.NewString())
	rotated, err := store.RotateOAuthRefreshToken(ctx, id, refresh,
		nextAccess, nextRefresh, time.Now().Add(time.Minute))
	if err != nil || rotated.AccountID != accountID || rotated.Scope != "profile:read" {
		t.Fatalf("legitimate rotation failed: %+v %v", rotated, err)
	}
	if _, err := store.RotateOAuthRefreshToken(ctx, id, refresh,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()),
		time.Now().Add(time.Minute)); !errors.Is(err, ErrOAuthGrantReplay) {
		t.Fatalf("refresh replay was not detected: %v", err)
	}
	if _, err := store.ResolveOAuthAccessToken(ctx, nextAccess); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("replay did not revoke current access token: %v", err)
	}
	if _, err := store.RotateOAuthRefreshToken(ctx, id, nextRefresh,
		hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()),
		time.Now().Add(time.Minute)); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("revoked family refresh token accepted: %v", err)
	}
}

func TestOAuthAuthorizationCodeConcurrentExchange(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	id := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: id, OwnerAccountID: accountID, Name: "concurrency test", RedirectUri: redirect,
	}); err != nil {
		t.Fatal(err)
	}
	code := hashOAuthTest(uuid.NewString())
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	if err := issueOAuthCodeForTest(t, ctx, store, id, accountID, redirect,
		"profile:read", challenge, code, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	const workers = 8
	var wg sync.WaitGroup
	successes := make(chan struct{}, workers)
	errorsSeen := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.RedeemOAuthAuthCode(ctx, id, code, redirect, challenge,
				hashOAuthTest(uuid.NewString()), hashOAuthTest(uuid.NewString()),
				time.Now().Add(time.Minute), time.Now().Add(time.Hour), uuid.New())
			switch {
			case err == nil:
				successes <- struct{}{}
			case errors.Is(err, ErrOAuthGrantInvalid):
			default:
				errorsSeen <- err
			}
		}()
	}
	wg.Wait()
	if len(errorsSeen) != 0 {
		t.Fatalf("unexpected concurrency database error: %v", <-errorsSeen)
	}
	if len(successes) != 1 {
		t.Fatalf("authorization code redeemed %d times, want once", len(successes))
	}
}
