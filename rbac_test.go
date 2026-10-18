package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type rbacFixture struct {
	pool *pgxpool.Pool
	auth *auth.Service
	org *organization.Service
	source *source.Service
	owner, member, orgID, otherOrgID, membership, primary int64
	ownerToken, memberToken, strangerToken string
}

func setupRBAC(t *testing.T) rbacFixture {
	t.Helper()
	p := coreDB(t)
	a := auth.New(p, nil)
	o := organization.New(p)
	s := source.NewService(p)
	ctx := context.Background()
	reviewer, _ := coreUser(t, a, "rbac-reviewer@example.test")
	owner, ownerToken := coreUser(t, a, "rbac-owner@example.test")
	member, memberToken := coreUser(t, a, "rbac-member@example.test")
	_, strangerToken := coreUser(t, a, "rbac-stranger@example.test")
	if _, err := p.Exec(ctx, "INSERT INTO platform_organization_reviewers(account_id,authorization_reference) VALUES($1,'test approval')", reviewer); err != nil { t.Fatal(err) }
	reg, err := o.SubmitRegistration(ctx, owner, "RBAC Realty", "VN", "RBAC-01")
	if err != nil { t.Fatal(err) }
	orgID, err := o.ApproveRegistration(ctx, reviewer, reg, "registry checked")
	if err != nil { t.Fatal(err) }
	otherReg, err := o.SubmitRegistration(ctx, reviewer, "Other Realty", "VN", "RBAC-02")
	if err != nil { t.Fatal(err) }
	otherOrgID, err := o.ApproveRegistration(ctx, reviewer, otherReg, "registry checked")
	if err != nil { t.Fatal(err) }
	var role, primary int64
	if err := p.QueryRow(ctx, "SELECT id FROM organization_roles WHERE organization_id=$1 AND name='Workspace Administrator'", orgID).Scan(&role); err != nil { t.Fatal(err) }
	if err := p.QueryRow(ctx, "SELECT membership_id FROM organization_primary_administrators WHERE organization_id=$1", orgID).Scan(&primary); err != nil { t.Fatal(err) }
	invitation, err := o.Invite(ctx, owner, orgID, role, "rbac-member@example.test")
	if err != nil { t.Fatal(err) }
	membership, err := o.AcceptInvitation(ctx, member, invitation)
	if err != nil { t.Fatal(err) }
	return rbacFixture{pool:p,auth:a,org:o,source:s,owner:owner,member:member,
		orgID:orgID,otherOrgID:otherOrgID,membership:membership,primary:primary,
		ownerToken:ownerToken,memberToken:memberToken,strangerToken:strangerToken}
}

func (f rbacFixture) request(method, path, token, key, body string) *httptest.ResponseRecorder {
	srv := api.NewServer(f.pool, f.auth, f.org, f.source, true, "")
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" { req.Header.Set("Content-Type", "application/json") }
	if token != "" { req.AddCookie(&http.Cookie{Name:"bdspro_session",Value:token}) }
	if key != "" { req.Header.Set("Idempotency-Key", key) }
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w,req)
	return w
}

func TestRBACRemoveMemberDoesNotInvalidateSession(t *testing.T) {
	f := setupRBAC(t)
	ctx := context.Background()
	existing, err := f.source.Capture(ctx,f.member,f.orgID,"prior-record",source.CaptureInput{Channel:"zalo"})
	if err != nil { t.Fatal(err) }
	route := fmt.Sprintf("/organizations/%d/members/%d",f.orgID,f.membership)
	if w:=f.request(http.MethodDelete,route,"","","");w.Code!=401 { t.Fatalf("anonymous=%d",w.Code) }
	if w:=f.request(http.MethodDelete,route,f.strangerToken,"","");w.Code!=404 { t.Fatalf("stranger=%d",w.Code) }
	if w:=f.request(http.MethodDelete,fmt.Sprintf("/organizations/%d/members/%d",f.otherOrgID,f.membership),f.ownerToken,"","");w.Code!=404 { t.Fatalf("cross tenant=%d",w.Code) }
	if w:=f.request(http.MethodDelete,fmt.Sprintf("/organizations/%d/members/%d",f.orgID,f.primary),f.ownerToken,"","");w.Code!=404 { t.Fatalf("primary protected=%d",w.Code) }
	if w:=f.request(http.MethodDelete,route,f.ownerToken,"","");w.Code!=204 { t.Fatalf("remove=%d %s",w.Code,w.Body.String()) }
	if w:=f.request(http.MethodDelete,route,f.ownerToken,"","");w.Code!=404 { t.Fatalf("repeat=%d",w.Code) }
	var openMemberships,openRoles int
	if err:=f.pool.QueryRow(ctx,"SELECT count(*) FROM organization_memberships WHERE id=$1 AND ended_at IS NULL",f.membership).Scan(&openMemberships);err!=nil { t.Fatal(err) }
	if err:=f.pool.QueryRow(ctx,"SELECT count(*) FROM organization_role_assignments WHERE membership_id=$1 AND ended_at IS NULL",f.membership).Scan(&openRoles);err!=nil { t.Fatal(err) }
	if openMemberships!=0 || openRoles!=0 { t.Fatalf("non-atomic revoke: %d memberships / %d roles",openMemberships,openRoles) }
	if w:=f.request(http.MethodGet,"/auth/me",f.memberToken,"","");w.Code!=200 { t.Fatalf("session wrongly revoked=%d",w.Code) }
	sourcePath:=fmt.Sprintf("/organizations/%d/sources",f.orgID)
	if w:=f.request(http.MethodPost,sourcePath,f.memberToken,"after-revoke",`{"channel":"zalo"}`);w.Code!=404 { t.Fatalf("write after revoke=%d",w.Code) }
	if w:=f.request(http.MethodGet,fmt.Sprintf("%s/%d",sourcePath,existing.ID),f.memberToken,"","");w.Code!=404 { t.Fatalf("read after revoke=%d",w.Code) }
	if w:=f.request(http.MethodGet,fmt.Sprintf("%s/%d",sourcePath,existing.ID),f.ownerToken,"","");w.Code!=200 { t.Fatalf("owner read=%d",w.Code) }
}

func TestRBACSourceWriterLockBlocksMemberRemoval(t *testing.T) {
	f:=setupRBAC(t)
	ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second)
	defer cancel()
	tx,err:=f.pool.Begin(ctx)
	if err!=nil { t.Fatal(err) }
	defer tx.Rollback(context.Background())
	if _,err=dbsqlc.New(tx).LockOrganizationForSourceWrite(ctx,f.orgID);err!=nil { t.Fatal(err) }
	other,err:=f.pool.Begin(ctx)
	if err!=nil { t.Fatal(err) }
	var id int64
	err=other.QueryRow(ctx,"SELECT party_id FROM organizations WHERE party_id=$1 FOR UPDATE NOWAIT",f.orgID).Scan(&id)
	var pgErr *pgconn.PgError
	if !errors.As(err,&pgErr)||pgErr.Code!="55P03" { t.Fatalf("exclusive lock bypassed active writer: %v",err) }
	_ = other.Rollback(context.Background())
	done:=make(chan error,1)
	go func(){ done<-f.org.RemoveMember(ctx,f.owner,f.orgID,f.membership) }()
	select {
	case e:=<-done: t.Fatalf("removal overtook writer: %v",e)
	case <-time.After(70*time.Millisecond):
	}
	if err:=tx.Commit(ctx);err!=nil { t.Fatal(err) }
	select {
	case e:=<-done: if e!=nil { t.Fatalf("remove failed: %v",e) }
	case <-ctx.Done(): t.Fatalf("removal stalled: %v",ctx.Err())
	}
	_,err=f.source.Capture(ctx,f.member,f.orgID,"blocked-after-remove",source.CaptureInput{Channel:"zalo"})
	if !errors.Is(err,source.ErrForbidden) { t.Fatalf("revoked user wrote source: %v",err) }
}
