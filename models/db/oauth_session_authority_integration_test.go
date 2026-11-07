package db

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Revocation and authorization must serialize on the same sessions row.
// A revoked or otherwise inactive session cannot change grants or create code.
func TestOAuthCodeIssuanceRejectsRevokedAndUnboundSession(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	sessionID := createOAuthTestSession(t, ctx, store, accountID)
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _, err := store.CreateOAuthClient(ctx, OauthClient{
		ID: clientID, OwnerAccountID: accountID,
		Name: "revoked session test", RedirectUri: redirect,
	}); err != nil { t.Fatal(err) }
	challenge := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	authorize := func(session uuid.UUID, code []byte) error {
		return store.ApproveOAuthAuthCode(ctx, clientID, accountID, redirect,
			"profile:read", challenge, code, time.Now().Add(time.Minute), 0, session)
	}
	if err := authorize(uuid.Nil, hashOAuthTest(uuid.NewString())); err == nil {
		t.Fatal("unbound access JWT created an OAuth grant")
	}
	if _, err := store.pool.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1",sessionID); err != nil {
		t.Fatal(err)
	}
	code := hashOAuthTest(uuid.NewString())
	if err := authorize(sessionID,code); !errors.Is(err,ErrOAuthSessionInactive) {
		t.Fatalf("revoked session issued code: %v",err)
	}
	var codes int
	if err:=store.pool.QueryRow(ctx,"SELECT count(*) FROM oauth_auth_codes WHERE code_hash=$1",code).Scan(&codes);err!=nil||codes!=0 {
		t.Fatalf("revoked session persisted an OAuth code: %d %v",codes,err)
	}
	version,err:=store.GetOAuthConsentVersion(ctx,accountID,clientID)
	if err!=nil||version!=0 {t.Fatalf("rejected authorization modified grant version=%d, %v",version,err)}
}

func TestOAuthCodeIssuanceCannotRaceSessionRevocation(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	sessionID := createOAuthTestSession(t, ctx, store, accountID)
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _,err:=store.CreateOAuthClient(ctx,OauthClient{
		ID:clientID,OwnerAccountID:accountID,Name:"session lock test",RedirectUri:redirect,
	});err!=nil {t.Fatal(err)}
	// Hold the same UPDATE row lock a successful session revocation uses.
	tx,err:=store.pool.Begin(ctx)
	if err!=nil {t.Fatal(err)}
	defer tx.Rollback(ctx)
	if _,err:=tx.Exec(ctx,"UPDATE sessions SET is_blocked=TRUE WHERE id=$1",sessionID);err!=nil {t.Fatal(err)}
	code:=hashOAuthTest(uuid.NewString())
	issued:=make(chan error,1)
	go func() {
		issued<-store.ApproveOAuthAuthCode(ctx,clientID,accountID,redirect,
			"profile:read","abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ",
			code,time.Now().Add(time.Minute),0,sessionID)
	}()
	// Without FOR SHARE the authorization would observe the pre-revoke row
	// and commit a new grant even while the revoke transaction is outstanding.
	select {
	case err:=<-issued:
		t.Fatalf("authorization did not wait for session row lock: %v",err)
	case <-time.After(150*time.Millisecond):
	}
	if err:=tx.Commit(ctx);err!=nil {t.Fatal(err)}
	select {
	case err:=<-issued:
		if !errors.Is(err,ErrOAuthSessionInactive) {
			t.Fatalf("authorization succeeded after revoke commit: %v",err)
		}
	case <-time.After(3*time.Second):
		t.Fatal("authorization deadlocked against session revocation")
	}
	var codes int
	if err:=store.pool.QueryRow(ctx,"SELECT count(*) FROM oauth_auth_codes WHERE code_hash=$1",code).Scan(&codes);err!=nil||codes!=0 {
		t.Fatalf("revoked session persisted code: %d %v",codes,err)
	}
}

func TestOAuthCodeIssuanceRejectsExpiredSession(t *testing.T) {
	ctx, store, accountID := oauthTestStore(t)
	sessionID := createOAuthTestSession(t, ctx, store, accountID)
	clientID := uuid.New()
	redirect := "https://client.example.test/callback"
	if _,err:=store.CreateOAuthClient(ctx,OauthClient{
		ID:clientID,OwnerAccountID:accountID,Name:"session expiry test",RedirectUri:redirect,
	});err!=nil {t.Fatal(err)}
	// Expiry is not revocation, but must equally prevent code issuance.
	if _,err:=store.pool.Exec(ctx,
		"UPDATE sessions SET expires_at=now()-interval '1 second' WHERE id=$1",sessionID);err!=nil {t.Fatal(err)}
	if err:=store.ApproveOAuthAuthCode(ctx,clientID,accountID,redirect,
		"profile:read","abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ",
		hashOAuthTest(uuid.NewString()),time.Now().Add(time.Minute),0,sessionID);
		!errors.Is(err,ErrOAuthSessionInactive) {
		t.Fatalf("expired session issued OAuth code: %v",err)
	}
}
