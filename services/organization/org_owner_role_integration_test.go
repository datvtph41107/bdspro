package organization

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ownerRoleCase struct {
	teamCase
	recipientAccountID int64
	recipientSessionID uuid.UUID
	actorPersonID int64
}

func newOwnerRoleCase(t *testing.T, ctx context.Context, pool *pgxpool.Pool, store db.Store, verifyTarget bool) ownerRoleCase {
	t.Helper()
	d := newTeamCase(t, ctx, store)
	var ownerPersonID int64
	if err := pool.QueryRow(ctx,"SELECT person_id FROM accounts WHERE id=$1",d.accountID).Scan(&ownerPersonID);err!=nil {
		t.Fatal(err)
	}
	recipientID:=accountIDForPerson(t,ctx,pool,d.recipientPersonID)
	if verifyTarget {
		recipient,err:=store.GetAccountByID(ctx,recipientID)
		if err!=nil {t.Fatal(err)}
		if _,err:=store.MarkEmailVerified(ctx,db.MarkEmailVerifiedParams{ID:recipient.ID,Email:recipient.Email});err!=nil {t.Fatal(err)}
	}
	sid:=uuid.New()
	if _,err:=store.CreateSession(ctx,db.CreateSessionParams{
		ID:sid,AccountID:recipientID,RefreshToken:"owner-role-"+uuid.NewString(),
		UserAgent:"G1.2c.8b",ClientIp:"203.0.113.31",ExpiresAt:time.Now().Add(time.Hour),
	});err!=nil {t.Fatal(err)}
	return ownerRoleCase{teamCase:d,recipientAccountID:recipientID,recipientSessionID:sid,actorPersonID:ownerPersonID}
}

func requireRole(t *testing.T,ctx context.Context,pool *pgxpool.Pool,orgID,personID int64,want string) {
	t.Helper()
	var role string
	err:=pool.QueryRow(ctx,`SELECT role FROM organization_members
		WHERE organization_id=$1 AND person_id=$2 AND ended_at IS NULL`,orgID,personID).Scan(&role)
	if err!=nil||role!=want {t.Fatalf("org=%d person=%d role=%q expected=%q err=%v",orgID,personID,role,want,err)}
}

func requireOwnerCount(t *testing.T,ctx context.Context,pool *pgxpool.Pool,orgID int64,want int) {
	t.Helper()
	var count int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM organization_members WHERE organization_id=$1 AND role='owner' AND ended_at IS NULL`,orgID).Scan(&count);err!=nil||count!=want {
		t.Fatalf("organization owner count %d expected %d err=%v",count,want,err)
	}
}

func TestOrganizationOwnerRoleLifecycleAndAudit(t *testing.T) {
	ctx,pool,store:=membershipTestDB(t)
	d:=newOwnerRoleCase(t,ctx,pool,store,true)
	if err:=DemoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.actorPersonID);!errors.Is(err,org_model.ErrLastOrganizationOwner) {
		t.Fatalf("sole owner demoted: %v",err)
	}
	if _,err:=PromoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);err!=nil {t.Fatal(err)}
	requireOwnerCount(t,ctx,pool,d.orgID,2)
	if _,err:=PromoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);!errors.Is(err,org_model.ErrOwnerCandidateInvalid) {
		t.Fatalf("double promotion accepted: %v",err)
	}
	if err:=DemoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.actorPersonID);err!=nil {t.Fatal(err)}
	requireRole(t,ctx,pool,d.orgID,d.actorPersonID,org_model.RoleMember)
	if _,err:=PromoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.actorPersonID);!errors.Is(err,org_model.ErrOrganizationOwnerRequired) {
		t.Fatalf("former owner still authorized: %v",err)
	}
	if _,err:=TransferOrganizationOwner(ctx,store,d.recipientAccountID,d.recipientSessionID,d.orgID,d.actorPersonID);err!=nil {t.Fatal(err)}
	requireRole(t,ctx,pool,d.orgID,d.actorPersonID,org_model.RoleOwner)
	requireRole(t,ctx,pool,d.orgID,d.recipientPersonID,org_model.RoleMember)
	requireOwnerCount(t,ctx,pool,d.orgID,1)
	for _,action:=range []string{"organization.owner.promoted","organization.owner.demoted","organization.owner.transferred"}{
		var count int
		if err:=pool.QueryRow(ctx,`SELECT count(*) FROM audit_events WHERE action=$1
			AND actor_account_id>0 AND scope_type='organization' AND scope_ref=$2`,
			action,fmt.Sprint(d.orgID)).Scan(&count);err!=nil||count!=1 {
			t.Fatalf("audit %s count=%d err=%v",action,count,err)
		}
	}
}

func TestOrganizationOwnerCandidateVerificationAndRollback(t *testing.T) {
	ctx,pool,store:=membershipTestDB(t)
	d:=newOwnerRoleCase(t,ctx,pool,store,false)
	if _,err:=TransferOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);!errors.Is(err,org_model.ErrOwnerCandidateInvalid) {
		t.Fatalf("unverified candidate acquired owner role: %v",err)
	}
	requireRole(t,ctx,pool,d.orgID,d.actorPersonID,org_model.RoleOwner)
	if _,err:=TransferOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,9223372036854775807);!errors.Is(err,org_model.ErrOwnerCandidateInvalid) {
		t.Fatalf("nonmember acquired owner: %v",err)
	}
	if _,err:=TransferOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.actorPersonID);!errors.Is(err,org_model.ErrInvalidOrganization) {
		t.Fatalf("self-transfer accepted: %v",err)
	}
	if _,err:=PromoteOrganizationOwner(ctx,store,d.recipientAccountID,d.recipientSessionID,d.orgID,d.actorPersonID);!errors.Is(err,org_model.ErrOrganizationOwnerRequired) {
		t.Fatalf("ordinary member promoted owner: %v",err)
	}
	// No account is associated with this Person; it must not be owner eligible.
	orphan,err:=db.WithTxResult(ctx,store,func(q *db.Queries)(int64,error){return db.CreatePersonWithParty(ctx,q)})
	if err!=nil {t.Fatal(err)}
	if _,err:=org_model.AddOrgUser(ctx,store,d.accountID,d.orgID,orphan);err!=nil {t.Fatal(err)}
	if _,err:=PromoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,orphan);!errors.Is(err,org_model.ErrOwnerCandidateInvalid) {
		t.Fatalf("orphan person made owner: %v",err)
	}
	// Same semantics for a phone-only verified recipient.
	phone:=fmt.Sprintf("+849%09d",d.recipientAccountID)
	if _,err:=pool.Exec(ctx,"INSERT INTO phone_credentials (account_id,phone_e164) VALUES ($1,$2)",d.recipientAccountID,phone);err!=nil {t.Fatal(err)}
	if _,err:=PromoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);err!=nil {
		t.Fatalf("verified phone owner candidate denied: %v",err)
	}
	// A failed audit must restore *both* roles, not just the last SQL update.
	broken:=auditRejectStore{Store:store,pool:pool}
	if _,err:=TransferOrganizationOwner(ctx,broken,d.accountID,d.sessionID,d.orgID,orphan);!errors.Is(err,org_model.ErrOwnerCandidateInvalid) {
		t.Fatalf("invalid candidate should be checked before audit: %v",err)
	}
	if _,err:=TransferOrganizationOwner(ctx,broken,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);!errors.Is(err,org_model.ErrOwnerCandidateInvalid) {
		t.Fatalf("already owner cannot be transferred ownership: %v",err)
	}
	// First demote the recipient while retaining original owner; test audit
	// rollback by then trying a full transfer with recipient ordinary again.
	if err:=DemoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);err!=nil {t.Fatal(err)}
	if _,err:=TransferOrganizationOwner(ctx,broken,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);!errors.Is(err,errAuditUnavailable) {
		t.Fatalf("audit failure did not roll back transfer: %v",err)
	}
	requireRole(t,ctx,pool,d.orgID,d.actorPersonID,org_model.RoleOwner)
	requireRole(t,ctx,pool,d.orgID,d.recipientPersonID,org_model.RoleMember)
	if err:=RemoveOrganizationMember(ctx,store,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);err!=nil {t.Fatal(err)}
	if _,err:=TransferOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);!errors.Is(err,org_model.ErrOwnerCandidateInvalid) {
		t.Fatalf("ended member became owner: %v",err)
	}
	requireOwnerCount(t,ctx,pool,d.orgID,1)
}

func TestConcurrentOrganizationOwnerDemotionsPreserveLastOwner(t *testing.T) {
	ctx,pool,store:=membershipTestDB(t)
	d:=newOwnerRoleCase(t,ctx,pool,store,true)
	if _,err:=PromoteOrganizationOwner(ctx,store,d.accountID,d.sessionID,d.orgID,d.recipientPersonID);err!=nil {t.Fatal(err)}
	blocker,err:=pool.Acquire(ctx);if err!=nil {t.Fatal(err)};defer blocker.Release()
	wa,err:=pool.Acquire(ctx);if err!=nil {t.Fatal(err)};defer wa.Release()
	wb,err:=pool.Acquire(ctx);if err!=nil {t.Fatal(err)};defer wb.Release()
	tx,err:=blocker.Begin(ctx);if err!=nil {t.Fatal(err)};defer tx.Rollback(ctx)
	var orgID int64
	if err:=tx.QueryRow(ctx,"SELECT party_id FROM organizations WHERE party_id=$1 FOR NO KEY UPDATE",d.orgID).Scan(&orgID);err!=nil {t.Fatal(err)}
	doneA,doneB:=make(chan error,1),make(chan error,1)
	go func(){doneA<-DemoteOrganizationOwner(ctx,teamPinnedStore{Store:store,conn:wa},d.accountID,d.sessionID,d.orgID,d.actorPersonID)}()
	go func(){doneB<-DemoteOrganizationOwner(ctx,teamPinnedStore{Store:store,conn:wb},d.recipientAccountID,d.recipientSessionID,d.orgID,d.recipientPersonID)}()
	waitTeamBlocking(t,ctx,pool,blocker,wa,doneA)
	waitTeamBlocking(t,ctx,pool,blocker,wb,doneB)
	if err:=tx.Commit(ctx);err!=nil {t.Fatal(err)}
	var first,second error
	select{case first=<-doneA:case <-time.After(5*time.Second):t.Fatal("first demotion stuck")}
	select{case second=<-doneB:case <-time.After(5*time.Second):t.Fatal("second demotion stuck")}
	success,denied:=0,0
	for _,e:=range []error{first,second}{
		if e==nil {success++}else if errors.Is(e,org_model.ErrLastOrganizationOwner){denied++}else{t.Fatalf("unexpected concurrent demotion: %v",e)}
	}
	if success!=1||denied!=1 {t.Fatalf("two demotions: success=%d last_owner=%d",success,denied)}
	requireOwnerCount(t,ctx,pool,d.orgID,1)
}

func TestOwnerTransferSerializesWithTeamGrant(t *testing.T) {
	ctx,pool,store:=membershipTestDB(t)
	d:=newOwnerRoleCase(t,ctx,pool,store,true)
	// Hold recipient's account row while a transfer has already demoted its
	// donor in an uncommitted transaction; the team writer must then wait for
	// the donor's owner membership, not acquire it in inverse order.
	blocker,err:=pool.Acquire(ctx);if err!=nil {t.Fatal(err)};defer blocker.Release()
	transferConn,err:=pool.Acquire(ctx);if err!=nil {t.Fatal(err)};defer transferConn.Release()
	teamConn,err:=pool.Acquire(ctx);if err!=nil {t.Fatal(err)};defer teamConn.Release()
	tx,err:=blocker.Begin(ctx);if err!=nil {t.Fatal(err)};defer tx.Rollback(ctx)
	var accountID int64
	if err:=tx.QueryRow(ctx,"SELECT id FROM accounts WHERE id=$1 FOR UPDATE",d.recipientAccountID).Scan(&accountID);err!=nil {t.Fatal(err)}
	transferDone:=make(chan error,1)
	go func(){
		_,e:=TransferOrganizationOwner(ctx,teamPinnedStore{Store:store,conn:transferConn},d.accountID,d.sessionID,d.orgID,d.recipientPersonID)
		transferDone<-e
	}()
	waitTeamBlocking(t,ctx,pool,blocker,transferConn,transferDone)
	teamDone:=make(chan error,1)
	go func(){
		_,e:=db.New(teamConn).AddTeamUserForOwner(ctx,db.AddTeamUserForOwnerParams{
			TeamID:d.teamID,ActorAccountID:d.accountID,PersonID:d.recipientPersonID,
		})
		teamDone<-e
	}()
	waitTeamBlocking(t,ctx,pool,transferConn,teamConn,teamDone)
	if err:=tx.Commit(ctx);err!=nil {t.Fatal(err)}
	select{case err:=<-transferDone:if err!=nil {t.Fatalf("transfer blocked by team writer: %v",err)};case <-time.After(5*time.Second):t.Fatal("transfer stuck")}
	select{case err:=<-teamDone:if !errors.Is(err,pgx.ErrNoRows){t.Fatalf("former owner authorized team grant: %v",err)};case <-time.After(5*time.Second):t.Fatal("team grant deadlocked")}
	requireRole(t,ctx,pool,d.orgID,d.actorPersonID,org_model.RoleMember)
	requireRole(t,ctx,pool,d.orgID,d.recipientPersonID,org_model.RoleOwner)
	requireOwnerCount(t,ctx,pool,d.orgID,1)
}

func TestOwnerTransferCannotOutliveRevokedActorSession(t *testing.T) {
	ctx,pool,store:=membershipTestDB(t)
	d:=newOwnerRoleCase(t,ctx,pool,store,true)
	revoker,err:=pool.Acquire(ctx);if err!=nil {t.Fatal(err)};defer revoker.Release()
	writer,err:=pool.Acquire(ctx);if err!=nil {t.Fatal(err)};defer writer.Release()
	tx,err:=revoker.Begin(ctx);if err!=nil {t.Fatal(err)};defer tx.Rollback(ctx)
	if _,err:=tx.Exec(ctx,"UPDATE sessions SET is_blocked=TRUE WHERE id=$1",d.sessionID);err!=nil {t.Fatal(err)}
	done:=make(chan error,1)
	go func(){_,e:=TransferOrganizationOwner(ctx,teamPinnedStore{Store:store,conn:writer},d.accountID,d.sessionID,d.orgID,d.recipientPersonID);done<-e}()
	waitTeamBlocking(t,ctx,pool,revoker,writer,done)
	if err:=tx.Commit(ctx);err!=nil {t.Fatal(err)}
	select{case e:=<-done:if !errors.Is(e,auth_model.ErrSessionUnavailable){t.Fatalf("revoked owner transferred: %v",e)};case <-time.After(5*time.Second):t.Fatal("session-locked transfer stuck")}
	requireRole(t,ctx,pool,d.orgID,d.actorPersonID,org_model.RoleOwner)
	requireRole(t,ctx,pool,d.orgID,d.recipientPersonID,org_model.RoleMember)
}
