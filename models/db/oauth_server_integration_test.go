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

func TestOAuthDirectSQLUsesContextOwnedTransaction(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	parent := WithStore(ctx, store)
	clientID := uuid.New()
	redirect := "https://direct-tx.example.test/callback"
	accessHash := hashOAuthTest(uuid.NewString())
	abort := errors.New("rollback raw OAuth methods with their owner")

	err := WithTx(parent, func(txCtx context.Context) error {
		created, err := store.CreateOAuthClient(txCtx, OauthClient{
			ID: clientID, OwnerAccountID: accountID,
			Name: "transactional OAuth client", RedirectUri: redirect,
		})
		if err != nil || created.ID != clientID {
			t.Fatalf("OAuth raw QueryRow did not use owner: client=%+v err=%v", created, err)
		}
		if got, err := store.GetOAuthClient(txCtx, clientID); err != nil || got.ID != clientID {
			t.Fatalf("OAuth client unreadable in owner transaction: %+v %v", got, err)
		}
		clients, err := store.ListOAuthClients(txCtx, accountID)
		if err != nil || len(clients) != 1 || clients[0].ID != clientID {
			t.Fatalf("OAuth raw Query did not use owner: %+v %v", clients, err)
		}
		if _, err := store.GetOAuthClient(ctx, clientID); !errors.Is(err, ErrOAuthClientMissing) {
			t.Fatalf("uncommitted OAuth client visible outside owner: %v", err)
		}

		dbx, err := store.queryExecutor(txCtx)
		if err != nil {
			return err
		}
		if _, err := dbx.Exec(txCtx, `INSERT INTO oauth_grants
			(account_id,client_id,scope,version,approved_at)
			VALUES($1,$2,'profile:read',7,now())`, accountID, clientID); err != nil {
			return err
		}
		version, err := store.GetOAuthConsentVersion(txCtx, accountID, clientID)
		if err != nil || version != 7 {
			t.Fatalf("OAuth consent preview did not read owning tx: version=%d err=%v", version, err)
		}
		authorizations, err := store.ListOAuthAuthorizations(txCtx, accountID)
		if err != nil || len(authorizations) != 1 || authorizations[0].ClientID != clientID {
			t.Fatalf("OAuth authorization list ignored owner: %+v %v", authorizations, err)
		}
		if outside, err := store.GetOAuthConsentVersion(ctx, accountID, clientID); err != nil || outside != 0 {
			t.Fatalf("uncommitted OAuth consent visible outside owner: version=%d err=%v", outside, err)
		}
		if _, err := dbx.Exec(txCtx, `INSERT INTO oauth_access_tokens
			(token_hash,client_id,account_id,scope,expires_at)
			VALUES($1,$2,$3,'profile:read',$4)`, accessHash, clientID, accountID, time.Now().Add(time.Hour)); err != nil {
			return err
		}
		access, err := store.ResolveOAuthAccessToken(txCtx, accessHash)
		if err != nil || access.AccountID != accountID || access.Scope != "profile:read" {
			t.Fatalf("OAuth bearer read ignored owner: %+v %v", access, err)
		}
		if _, err := store.ResolveOAuthAccessToken(ctx, accessHash); !errors.Is(err, ErrOAuthGrantInvalid) {
			t.Fatalf("uncommitted bearer token visible outside owner: %v", err)
		}
		return abort
	})
	if !errors.Is(err, abort) {
		t.Fatalf("OAuth parent rollback returned %v", err)
	}
	if _, err := store.GetOAuthClient(ctx, clientID); !errors.Is(err, ErrOAuthClientMissing) {
		t.Fatalf("OAuth client survived parent rollback: %v", err)
	}
	if _, err := store.ResolveOAuthAccessToken(ctx, accessHash); !errors.Is(err, ErrOAuthGrantInvalid) {
		t.Fatalf("OAuth bearer survived parent rollback: %v", err)
	}
}

func TestOAuthDirectRevocationRollsBackWithParent(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	clientID := uuid.New()
	if _, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: clientID, OwnerAccountID: accountID,
		Name: "rollbackable client", RedirectUri: "https://revoke.example.test/callback",
	}); err != nil {
		t.Fatal(err)
	}
	abort := errors.New("abort client revocation")
	err := WithTx(WithStore(ctx, store), func(txCtx context.Context) error {
		changed, err := store.RevokeOAuthClient(txCtx, accountID, clientID)
		if err != nil || !changed {
			t.Fatalf("OAuth raw Exec did not revoke in owner: changed=%v err=%v", changed, err)
		}
		if _, err := store.GetOAuthClient(txCtx, clientID); !errors.Is(err, ErrOAuthClientMissing) {
			t.Fatalf("owner still sees revoked client: %v", err)
		}
		if _, err := store.GetOAuthClient(ctx, clientID); err != nil {
			t.Fatalf("raw Exec revoked client outside transaction before commit: %v", err)
		}
		return abort
	})
	if !errors.Is(err, abort) {
		t.Fatalf("parent rollback returned %v", err)
	}
	if _, err := store.GetOAuthClient(ctx, clientID); err != nil {
		t.Fatalf("client revocation survived rollback: %v", err)
	}
}

func TestOAuthRawTransactionRefusesSQLCNesting(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	clientID := uuid.New()
	if _, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: clientID, OwnerAccountID: accountID,
		Name: "raw nesting owner", RedirectUri: "https://raw-nesting.example.test/callback",
	}); err != nil {
		t.Fatal(err)
	}

	var uncommittedParty int64
	err := WithTx(WithStore(ctx, store), func(txCtx context.Context) error {
		var err error
		uncommittedParty, err = store.CreateParty(txCtx)
		if err != nil {
			return err
		}
		// This OAuth operation intentionally owns a raw transaction.
		// Until replay/revocation commit semantics are explicitly designed
		// for nesting, it must refuse to start a second transaction.
		innerErr := store.RevokeOAuthAuthorization(txCtx, accountID, clientID)
		if !errors.Is(innerErr, ErrRawTransactionInSQLC) {
			t.Fatalf("nested raw OAuth transaction was accepted: %v", innerErr)
		}
		return nil // A swallowed refusal must not commit earlier SQLC writes.
	})
	if !errors.Is(err, ErrRawTransactionInSQLC) {
		t.Fatalf("raw OAuth refusal did not poison outer transaction: %v", err)
	}
	var count int
	if err := store.pool.QueryRow(ctx,
		"SELECT count(*) FROM parties WHERE id=$1", uncommittedParty).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("outer SQLC write committed after nested raw OAuth refusal")
	}
	if _, err := store.GetOAuthClient(ctx, clientID); err != nil {
		t.Fatalf("nested OAuth refusal revoked client unexpectedly: %v", err)
	}
}
