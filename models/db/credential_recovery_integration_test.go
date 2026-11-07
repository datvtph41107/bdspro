package db_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type oauthRecoveryFixture struct {
	client uuid.UUID
	accountID int64
	session uuid.UUID
	access []byte
	refresh []byte
	pending []byte
}

func makeRecoveryFixture(t *testing.T, ctx context.Context, store db.Store, oauth db.OAuthServerStore, owner int64) oauthRecoveryFixture {
	t.Helper()
	account,err:=account_model.Register(ctx,store,account_model.RegisterParams{
		Email:uuid.NewString()+"@example.test",PasswordHash:"original-recovery-test-hash",
	})
	if err!=nil {t.Fatal(err)}
	session,err:=store.CreateSession(ctx,db.CreateSessionParams{
		ID:uuid.New(),AccountID:account.ID,RefreshToken:"recovery-"+uuid.NewString(),
		UserAgent:"recovery-test",ClientIp:"127.0.0.1",ExpiresAt:time.Now().Add(time.Hour),
	})
	if err!=nil {t.Fatal(err)}
	clientID:=uuid.New()
	const redirect="https://client.example.test/callback"
	if _,err:=oauth.CreateOAuthClient(ctx,db.OauthClient{
		ID:clientID,OwnerAccountID:owner,Name:"recovery-test-app",RedirectUri:redirect,
	});err!=nil {t.Fatal(err)}
	challenge:="abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
	code:=sha256.Sum256([]byte(uuid.NewString()))
	if err:=oauth.ApproveOAuthAuthCode(ctx,clientID,account.ID,redirect,"profile:read",
		challenge,code[:],time.Now().Add(time.Minute),0,session.ID);err!=nil {t.Fatal(err)}
	access:=sha256.Sum256([]byte(uuid.NewString()))
	refresh:=sha256.Sum256([]byte(uuid.NewString()))
	if _,err:=oauth.RedeemOAuthAuthCode(ctx,clientID,code[:],redirect,challenge,
		access[:],refresh[:],time.Now().Add(time.Minute),time.Now().Add(time.Hour),uuid.New());err!=nil {t.Fatal(err)}
	pending:=sha256.Sum256([]byte(uuid.NewString()))
	if err:=oauth.ApproveOAuthAuthCode(ctx,clientID,account.ID,redirect,"profile:read",
		challenge,pending[:],time.Now().Add(time.Minute),1,session.ID);err!=nil {t.Fatal(err)}
	return oauthRecoveryFixture{client:clientID,accountID:account.ID,session:session.ID,
		access:access[:],refresh:refresh[:],pending:pending[:]}
}

func resetRecoveryProof(t *testing.T,ctx context.Context,store db.Store,accountID int64) []byte {
	t.Helper()
	proof:=sha256.Sum256([]byte(uuid.NewString()))
	if err:=store.CreatePasswordResetToken(ctx,db.CreatePasswordResetTokenParams{
		TokenHash:proof[:],AccountID:accountID,
	});err!=nil {t.Fatal(err)}
	return proof[:]
}

func TestPasswordRecoveryAtomicallyRevokesOAuthAndPreservesUnrelatedAccounts(t *testing.T) {
	ctx,_,store:=authChannelsTestStore(t)
	oauth,ok:=store.(db.OAuthServerStore)
	if !ok {t.Fatal("SQLStore must implement OAuthServerStore")}
	owner,err:=account_model.Register(ctx,store,account_model.RegisterParams{
		Email:uuid.NewString()+"@example.test",PasswordHash:"owner-original",
	})
	if err!=nil {t.Fatal(err)}
	victim:=makeRecoveryFixture(t,ctx,store,oauth,owner.ID)
	other:=makeRecoveryFixture(t,ctx,store,oauth,owner.ID)
	proof:=resetRecoveryProof(t,ctx,store,victim.accountID)
	if err:=auth_model.ResetPassword(ctx,store,proof,"replacement-hash");err!=nil {t.Fatal(err)}
	password,err:=store.GetPasswordHash(ctx,victim.accountID)
	if err!=nil||password!="replacement-hash" {t.Fatalf("password not changed: %q, %v",password,err)}
	if _,err:=store.GetSession(ctx,victim.session);!errors.Is(err,pgx.ErrNoRows) {
		t.Fatalf("password reset left victim session: %v",err)
	}
	if _,err:=store.GetSession(ctx,other.session);err!=nil {t.Fatalf("recovery revoked other user's session: %v",err)}
	if _,err:=oauth.ResolveOAuthAccessToken(ctx,victim.access);!errors.Is(err,db.ErrOAuthGrantInvalid){
		t.Fatalf("victim OAuth access survived recovery: %v",err)
	}
	if _,err:=oauth.RotateOAuthRefreshToken(ctx,victim.client,victim.refresh,
		[]byte("01234567890123456789012345678901"),[]byte("abcd0123456789012345678901234567"),time.Now().Add(time.Minute));
		!errors.Is(err,db.ErrOAuthGrantInvalid) {t.Fatalf("victim refresh survived recovery: %v",err)}
	if _,err:=oauth.RedeemOAuthAuthCode(ctx,victim.client,victim.pending,
		"https://client.example.test/callback","abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ",
		[]byte("01234567890123456789012345678901"),[]byte("abcd0123456789012345678901234567"),
		time.Now().Add(time.Minute),time.Now().Add(time.Hour),uuid.New());
		!errors.Is(err,db.ErrOAuthGrantInvalid) {t.Fatalf("pending code survived recovery: %v",err)}
	mine,err:=oauth.ListOAuthAuthorizations(ctx,victim.accountID)
	if err!=nil||len(mine)!=0 {t.Fatalf("recovery retained authorization: %+v %v",mine,err)}
	if _,err:=oauth.ResolveOAuthAccessToken(ctx,other.access);err!=nil {t.Fatalf("other user's OAuth access revoked: %v",err)}
	if _,err:=oauth.GetOAuthClient(ctx,victim.client);err!=nil {t.Fatalf("client registration was incorrectly revoked: %v",err)}
	if err:=auth_model.ResetPassword(ctx,store,proof,"replayed-hash");!errors.Is(err,auth_model.ErrPasswordResetInvalid){
		t.Fatalf("replayed reset token accepted: %v",err)
	}
}

func TestRevokeAllFirstPartySessionsPreservesDelegatedOAuthGrants(t *testing.T) {
	ctx,_,store:=authChannelsTestStore(t)
	oauth:=store.(db.OAuthServerStore)
	owner,err:=account_model.Register(ctx,store,account_model.RegisterParams{
		Email:uuid.NewString()+"@example.test",PasswordHash:"owner-hash",
	})
	if err!=nil {t.Fatal(err)}
	fx:=makeRecoveryFixture(t,ctx,store,oauth,owner.ID)
	if err:=auth_model.RevokeAllSessions(ctx,store,fx.accountID);err!=nil {t.Fatal(err)}
	if _,err:=oauth.ResolveOAuthAccessToken(ctx,fx.access);err!=nil {
		t.Fatalf("internal session revoke unexpectedly revoked a third-party grant: %v",err)
	}
}

func TestPasswordRecoveryRollbackWhenOAuthRevokeFails(t *testing.T) {
	ctx,pool,store:=authChannelsTestStore(t)
	oauth:=store.(db.OAuthServerStore)
	owner,err:=account_model.Register(ctx,store,account_model.RegisterParams{
		Email:uuid.NewString()+"@example.test",PasswordHash:"owner-hash",
	})
	if err!=nil {t.Fatal(err)}
	fx:=makeRecoveryFixture(t,ctx,store,oauth,owner.ID)
	proof:=resetRecoveryProof(t,ctx,store,fx.accountID)
	blocker,err:=pool.Begin(ctx)
	if err!=nil {t.Fatal(err)}
	defer blocker.Rollback(ctx)
	var locked int64
	if err:=blocker.QueryRow(ctx,`SELECT version FROM oauth_grants WHERE account_id=$1 AND client_id=$2 FOR UPDATE`,
		fx.accountID,fx.client).Scan(&locked);err!=nil {t.Fatal(err)}
	short,cancel:=context.WithTimeout(ctx,300*time.Millisecond)
	defer cancel()
	if err:=auth_model.ResetPassword(short,store,proof,"replacement-hash");err==nil {
		t.Fatal("recovery unexpectedly completed through locked OAuth grant")
	}
	if err:=blocker.Rollback(ctx);err!=nil {t.Fatal(err)}
	current,err:=store.GetPasswordHash(ctx,fx.accountID)
	if err!=nil||current!="original-recovery-test-hash" {t.Fatalf("failed recovery modified password: %q %v",current,err)}
	if _,err:=store.GetSession(ctx,fx.session);err!=nil {t.Fatalf("failed recovery removed session: %v",err)}
	if _,err:=oauth.ResolveOAuthAccessToken(ctx,fx.access);err!=nil {t.Fatalf("failed recovery revoked OAuth early: %v",err)}
	if err:=auth_model.ResetPassword(ctx,store,proof,"replacement-hash");err!=nil {
		t.Fatalf("rollback consumed reset proof: %v",err)
	}
}

func TestOAuthRefreshCannotEscapeConcurrentPasswordRecovery(t *testing.T) {
	ctx,_,store:=authChannelsTestStore(t)
	oauth:=store.(db.OAuthServerStore)
	owner,err:=account_model.Register(ctx,store,account_model.RegisterParams{
		Email:uuid.NewString()+"@example.test",PasswordHash:"owner-hash",
	})
	if err!=nil {t.Fatal(err)}
	fx:=makeRecoveryFixture(t,ctx,store,oauth,owner.ID)
	proof:=resetRecoveryProof(t,ctx,store,fx.accountID)
	newAccess:=sha256.Sum256([]byte(uuid.NewString()))
	newRefresh:=sha256.Sum256([]byte(uuid.NewString()))
	start:=make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	var refreshErr,resetErr error
	go func(){defer wg.Done();<-start
		_,refreshErr=oauth.RotateOAuthRefreshToken(ctx,fx.client,fx.refresh,
			newAccess[:],newRefresh[:],time.Now().Add(time.Minute))
	}()
	go func(){defer wg.Done();<-start;resetErr=auth_model.ResetPassword(ctx,store,proof,"replacement-hash")}()
	close(start);wg.Wait()
	if resetErr!=nil {t.Fatalf("recovery failed: %v",resetErr)}
	if refreshErr!=nil&&!errors.Is(refreshErr,db.ErrOAuthGrantInvalid) {
		t.Fatalf("refresh returned unexpected error: %v",refreshErr)
	}
	for _,raw:=range [][]byte{fx.access,newAccess[:]} {
		if _,err:=oauth.ResolveOAuthAccessToken(ctx,raw);!errors.Is(err,db.ErrOAuthGrantInvalid){
			t.Fatalf("concurrent recovery left access active: %v",err)
		}
	}
}

func TestConcurrentPasswordRecoveryProofsOnOneAccountDoNotDeadlock(t *testing.T) {
	ctx,_,store:=authChannelsTestStore(t)
	account,err:=account_model.Register(ctx,store,account_model.RegisterParams{
		Email:uuid.NewString()+"@example.test",PasswordHash:"original-hash",
	})
	if err!=nil {t.Fatal(err)}
	first:=resetRecoveryProof(t,ctx,store,account.ID)
	second:=resetRecoveryProof(t,ctx,store,account.ID)
	start:=make(chan struct{})
	results:=make(chan error,2)
	var wg sync.WaitGroup
	for i,proof:=range [][]byte{first,second} {
		wg.Add(1)
		go func(i int, proof []byte) {
			defer wg.Done()
			<-start
			attemptCtx,cancel:=context.WithTimeout(ctx,5*time.Second)
			defer cancel()
			if i == 0 {
				results <- auth_model.ResetPassword(attemptCtx, store, proof, "first-password")
			} else {
				results <- auth_model.ResetPassword(attemptCtx, store, proof, "second-password")
			}
		}(i,proof)
	}
	close(start)
	wg.Wait()
	close(results)
	var accepted,denied int
	for err:=range results {
		switch {
		case err==nil:accepted++
		case errors.Is(err,auth_model.ErrPasswordResetInvalid):denied++
		default:t.Fatalf("concurrent proof error (possible deadlock): %v",err)
		}
	}
	if accepted!=1||denied!=1 {
		t.Fatalf("concurrent recovery accepted=%d denied=%d, want one of each",accepted,denied)
	}
}

func TestPasswordLoginSessionCannotSlipPastCommittedRecovery(t *testing.T) {
	base,pool,store:=authChannelsTestStore(t)
	ctx,cancel:=context.WithTimeout(base,10*time.Second)
	defer cancel()
	account,err:=account_model.Register(ctx,store,account_model.RegisterParams{
		Email:uuid.NewString()+"@example.test",PasswordHash:"verified-old-hash",
	})
	if err!=nil {t.Fatal(err)}
	blocker,err:=pool.Begin(ctx)
	if err!=nil {t.Fatal(err)}
	defer blocker.Rollback(ctx)
	var id int64
	if err:=blocker.QueryRow(ctx,"SELECT id FROM accounts WHERE id=$1 FOR UPDATE",account.ID).Scan(&id);err!=nil {t.Fatal(err)}
	sessionID:=uuid.New()
	finished:=make(chan error,1)
	go func(){
		_,err:=auth_model.NewSession(ctx,store,db.CreateSessionParams{
			ID:sessionID,AccountID:account.ID,RefreshToken:"login-"+uuid.NewString(),
			UserAgent:"old-login-race-test",ClientIp:"127.0.0.1",ExpiresAt:time.Now().Add(time.Hour),
		},"verified-old-hash")
		finished<-err
	}()
	select {
	case err:=<-finished:
		t.Fatalf("login session was inserted despite account recovery row lock: %v",err)
	case <-time.After(150*time.Millisecond):
	}
	if _,err:=blocker.Exec(ctx,"UPDATE password_credentials SET password_hash=$2 WHERE account_id=$1",
		account.ID,"recovered-new-hash");err!=nil {t.Fatal(err)}
	if err:=blocker.Commit(ctx);err!=nil {t.Fatal(err)}
	select {
	case err:=<-finished:
		if !errors.Is(err,auth_model.ErrInvalidCredentials) {
			t.Fatalf("old password hash produced a committed session after recovery: %v",err)
		}
	case <-time.After(3*time.Second):
		t.Fatal("login remained blocked after account recovery committed")
	}
	if _,err:=store.GetSession(ctx,sessionID);!errors.Is(err,pgx.ErrNoRows) {
		t.Fatalf("failed password proof persisted session: %v",err)
	}
	var histories int
	if err:=pool.QueryRow(ctx,"SELECT count(*) FROM account_connection_histories WHERE session_id=$1",
		sessionID).Scan(&histories);err!=nil||histories!=0 {
		t.Fatalf("failed password proof persisted connection history: %d %v",histories,err)
	}
}
