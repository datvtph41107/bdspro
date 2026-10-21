package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/datvtph41107/bdspro/token"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type rbacFixture struct {
	pool                                                  *pgxpool.Pool
	auth                                                  *auth.Service
	org                                                   *organization.Service
	source                                                *source.Service
	owner, member, orgID, otherOrgID, membership, primary int64
	ownerToken, memberToken, strangerToken                string
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
	if _, err := p.Exec(ctx, "INSERT INTO platform_organization_reviewers(account_id,authorization_reference) VALUES($1,'test approval')", reviewer); err != nil {
		t.Fatal(err)
	}
	reg, err := o.SubmitRegistration(ctx, owner, "RBAC Realty", "VN", "RBAC-01")
	if err != nil {
		t.Fatal(err)
	}
	orgID, err := o.ApproveRegistration(ctx, reviewer, reg, "registry checked")
	if err != nil {
		t.Fatal(err)
	}
	otherReg, err := o.SubmitRegistration(ctx, reviewer, "Other Realty", "VN", "RBAC-02")
	if err != nil {
		t.Fatal(err)
	}
	otherOrgID, err := o.ApproveRegistration(ctx, reviewer, otherReg, "registry checked")
	if err != nil {
		t.Fatal(err)
	}
	var role, primary int64
	if err := p.QueryRow(ctx, "SELECT id FROM organization_roles WHERE organization_id=$1 AND name='Workspace Administrator'", orgID).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if err := p.QueryRow(ctx, "SELECT membership_id FROM organization_primary_administrators WHERE organization_id=$1", orgID).Scan(&primary); err != nil {
		t.Fatal(err)
	}
	invitation, err := o.Invite(ctx, owner, orgID, role, "rbac-member@example.test")
	if err != nil {
		t.Fatal(err)
	}
	membership, err := o.AcceptInvitation(ctx, member, invitation)
	if err != nil {
		t.Fatal(err)
	}
	return rbacFixture{pool: p, auth: a, org: o, source: s, owner: owner, member: member,
		orgID: orgID, otherOrgID: otherOrgID, membership: membership, primary: primary,
		ownerToken: ownerToken, memberToken: memberToken, strangerToken: strangerToken}
}

func (f rbacFixture) request(method, path, token, key, body string) *httptest.ResponseRecorder {
	srv := api.NewServer(f.pool, f.auth, f.org, f.source, true, "")
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: token})
		req.Header.Set("X-BDSPro-Request", "1")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	return w
}

func TestRBACRemoveMemberDoesNotInvalidateSession(t *testing.T) {
	f := setupRBAC(t)
	ctx := context.Background()
	existing, err := f.source.Capture(ctx, f.member, f.orgID, "prior-record", source.CaptureInput{Channel: "zalo"})
	if err != nil {
		t.Fatal(err)
	}
	route := fmt.Sprintf("/organizations/%d/members/%d", f.orgID, f.membership)
	if w := f.request(http.MethodDelete, route, "", "", ""); w.Code != 401 {
		t.Fatalf("anonymous=%d", w.Code)
	}
	if w := f.request(http.MethodDelete, route, f.strangerToken, "", ""); w.Code != 404 {
		t.Fatalf("stranger=%d", w.Code)
	}
	if w := f.request(http.MethodDelete, fmt.Sprintf("/organizations/%d/members/%d", f.otherOrgID, f.membership), f.ownerToken, "", ""); w.Code != 404 {
		t.Fatalf("cross tenant=%d", w.Code)
	}
	if w := f.request(http.MethodDelete, fmt.Sprintf("/organizations/%d/members/%d", f.orgID, f.primary), f.ownerToken, "", ""); w.Code != 404 {
		t.Fatalf("primary protected=%d", w.Code)
	}
	if w := f.request(http.MethodDelete, route, f.ownerToken, "", ""); w.Code != 204 {
		t.Fatalf("remove=%d %s", w.Code, w.Body.String())
	}
	if w := f.request(http.MethodDelete, route, f.ownerToken, "", ""); w.Code != 404 {
		t.Fatalf("repeat=%d", w.Code)
	}
	var openMemberships, openRoles int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM organization_memberships WHERE id=$1 AND ended_at IS NULL", f.membership).Scan(&openMemberships); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM organization_role_assignments WHERE membership_id=$1 AND ended_at IS NULL", f.membership).Scan(&openRoles); err != nil {
		t.Fatal(err)
	}
	if openMemberships != 0 || openRoles != 0 {
		t.Fatalf("non-atomic revoke: %d memberships / %d roles", openMemberships, openRoles)
	}
	if w := f.request(http.MethodGet, "/auth/me", f.memberToken, "", ""); w.Code != 200 {
		t.Fatalf("session wrongly revoked=%d", w.Code)
	}
	sourcePath := fmt.Sprintf("/organizations/%d/sources", f.orgID)
	if w := f.request(http.MethodPost, sourcePath, f.memberToken, "after-revoke", `{"channel":"zalo"}`); w.Code != 404 {
		t.Fatalf("write after revoke=%d", w.Code)
	}
	if w := f.request(http.MethodGet, fmt.Sprintf("%s/%d", sourcePath, existing.ID), f.memberToken, "", ""); w.Code != 404 {
		t.Fatalf("read after revoke=%d", w.Code)
	}
	if w := f.request(http.MethodGet, fmt.Sprintf("%s/%d", sourcePath, existing.ID), f.ownerToken, "", ""); w.Code != 200 {
		t.Fatalf("owner read=%d", w.Code)
	}
	_, err = f.source.ReportPrice(ctx, f.member, f.orgID, existing.ID, "after-remove-price",
		source.PriceInput{Amount: "100", Currency: "VND"})
	if !errors.Is(err, source.ErrForbidden) {
		t.Fatalf("revoked price report: %v", err)
	}
}

func TestRBACSourceWriterLockBlocksMemberRemoval(t *testing.T) {
	f := setupRBAC(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = dbsqlc.New(tx).LockOrganizationForSourceWrite(ctx, f.orgID); err != nil {
		t.Fatal(err)
	}
	other, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	err = other.QueryRow(ctx, "SELECT party_id FROM organizations WHERE party_id=$1 FOR UPDATE NOWAIT", f.orgID).Scan(&id)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Fatalf("exclusive lock bypassed active writer: %v", err)
	}
	_ = other.Rollback(context.Background())
	done := make(chan error, 1)
	go func() { done <- f.org.RemoveMember(ctx, f.owner, f.orgID, f.membership) }()
	select {
	case e := <-done:
		t.Fatalf("removal overtook writer: %v", e)
	case <-time.After(70 * time.Millisecond):
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-done:
		if e != nil {
			t.Fatalf("remove failed: %v", e)
		}
	case <-ctx.Done():
		t.Fatalf("removal stalled: %v", ctx.Err())
	}
	_, err = f.source.Capture(ctx, f.member, f.orgID, "blocked-after-remove", source.CaptureInput{Channel: "zalo"})
	if !errors.Is(err, source.ErrForbidden) {
		t.Fatalf("revoked user wrote source: %v", err)
	}
}

func TestRBACInvitePermissionDoesNotGrantRemovePermission(t *testing.T) {
	f := setupRBAC(t)
	ctx := context.Background()
	// The invited member initially holds Workspace Administrator. Replace it
	// with a role granting invitation only: the scopes must remain distinct.
	_, err := f.pool.Exec(ctx, "UPDATE organization_role_assignments SET ended_at=now() WHERE membership_id=$1", f.membership)
	if err != nil {
		t.Fatal(err)
	}
	var inviteOnlyRole int64
	if err := f.pool.QueryRow(ctx,
		"INSERT INTO organization_roles(organization_id,name) VALUES($1,'Invitation Only') RETURNING id",
		f.orgID).Scan(&inviteOnlyRole); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx,
		"INSERT INTO organization_role_permissions(role_id,permission_code) VALUES($1,'members.invite')",
		inviteOnlyRole); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx,
		"INSERT INTO organization_role_assignments(organization_id,membership_id,role_id) VALUES($1,$2,$3)",
		f.orgID, f.membership, inviteOnlyRole); err != nil {
		t.Fatal(err)
	}
	if _, err := f.org.Invite(ctx, f.member, f.orgID, inviteOnlyRole, "rbac-stranger@example.test"); err != nil {
		t.Fatalf("members.invite permission must still work: %v", err)
	}
	if err := f.org.RemoveMember(ctx, f.member, f.orgID, f.membership); !errors.Is(err, organization.ErrNotFound) {
		t.Fatalf("invite-only actor unexpectedly removed a member: %v", err)
	}
	if err := f.org.RemoveMember(ctx, f.owner, f.orgID, f.membership); err != nil {
		t.Fatalf("actual tenant administrator cannot remove invite-only member: %v", err)
	}
}

func TestWorkspaceDiscoveryUsesLiveMembershipNotJWTClaims(t *testing.T) {
	f := setupRBAC(t)
	ctx := context.Background()

	read := func(token string) ([]organization.Workspace, int) {
		t.Helper()
		w := f.request(http.MethodGet, "/auth/workspaces", token, "", "")
		if w.Code != http.StatusOK {
			return nil, w.Code
		}
		var body struct {
			Workspaces []organization.Workspace `json:"workspaces"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Workspaces == nil {
			t.Fatal("workspaces must be [] instead of null")
		}
		return body.Workspaces, w.Code
	}
	if _, status := read(""); status != http.StatusUnauthorized {
		t.Fatalf("anonymous workspace status=%d", status)
	}
	for _, tc := range []struct {
		token string
		want  int64
		size  int
	}{
		{f.ownerToken, f.orgID, 1},
		{f.memberToken, f.orgID, 1},
		{f.strangerToken, 0, 0},
	} {
		rows, status := read(tc.token)
		if status != 200 || len(rows) != tc.size {
			t.Fatalf("workspace discovery status=%d rows=%+v, wanted=%d", status, rows, tc.size)
		}
		if tc.size == 1 && (rows[0].OrganizationID != tc.want || rows[0].MembershipID <= 0) {
			t.Fatalf("wrong workspace identity: %+v", rows)
		}
	}

	issuer, err := token.NewJWTMaker(strings.Repeat("W", 32))
	if err != nil {
		t.Fatal(err)
	}
	jwtToken, _, err := issuer.Issue(f.member, f.memberToken)
	if err != nil {
		t.Fatal(err)
	}
	srv := api.NewServer(f.pool, f.auth, f.org, f.source, true, "")
	srv.EnableJWTMaker(issuer)
	bearerRead := func() (int, []organization.Workspace) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/auth/workspaces", nil)
		request.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, request)
		if w.Code != 200 {
			return w.Code, nil
		}
		var body struct {
			Workspaces []organization.Workspace `json:"workspaces"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return w.Code, body.Workspaces
	}
	if status, rows := bearerRead(); status != 200 || len(rows) != 1 {
		t.Fatalf("bearer workspace status=%d rows=%+v", status, rows)
	}

	// The JWT stays signed and session stays valid, but a live block must
	// remove the workspace immediately from both cookie and Bearer listing.
	var blockID int64
	err = f.pool.QueryRow(ctx, "INSERT INTO organization_membership_blocks(membership_id) VALUES($1) RETURNING id", f.membership).Scan(&blockID)
	if err != nil {
		t.Fatal(err)
	}
	if rows, status := read(f.memberToken); status != 200 || len(rows) != 0 {
		t.Fatalf("blocked cookie user sees workspace: status=%d rows=%+v", status, rows)
	}
	if status, rows := bearerRead(); status != 200 || len(rows) != 0 {
		t.Fatalf("blocked Bearer user sees workspace: status=%d rows=%+v", status, rows)
	}
	if _, err := f.pool.Exec(ctx, "UPDATE organization_membership_blocks SET ended_at=now() WHERE id=$1", blockID); err != nil {
		t.Fatal(err)
	}
	if rows, status := read(f.memberToken); status != 200 || len(rows) != 1 {
		t.Fatalf("unblocked member cannot discover workspace: status=%d rows=%+v", status, rows)
	}

	// A membership with no active role/permission is not a usable workspace.
	if _, err := f.pool.Exec(ctx,
		"UPDATE organization_role_assignments SET ended_at=now() WHERE membership_id=$1",
		f.membership); err != nil {
		t.Fatal(err)
	}
	if rows, status := read(f.memberToken); status != 200 || len(rows) != 0 {
		t.Fatalf("role-less membership exposed: status=%d rows=%+v", status, rows)
	}
	var adminRole int64
	if err := f.pool.QueryRow(ctx,
		"SELECT id FROM organization_roles WHERE organization_id=$1 AND name='Workspace Administrator'",
		f.orgID).Scan(&adminRole); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx,
		"INSERT INTO organization_role_assignments(organization_id,membership_id,role_id) VALUES($1,$2,$3)",
		f.orgID, f.membership, adminRole); err != nil {
		t.Fatal(err)
	}

	if err := f.org.RemoveMember(ctx, f.owner, f.orgID, f.membership); err != nil {
		t.Fatal(err)
	}
	if rows, status := read(f.memberToken); status != 200 || len(rows) != 0 {
		t.Fatalf("removed member sees tenant: status=%d rows=%+v", status, rows)
	}
	if status, rows := bearerRead(); status != 200 || len(rows) != 0 {
		t.Fatalf("removed member JWT sees tenant: status=%d rows=%+v", status, rows)
	}
	if w := f.request(http.MethodGet, "/auth/me", f.memberToken, "", ""); w.Code != 200 {
		t.Fatalf("tenant removal revoked personal identity session: %d", w.Code)
	}

	if _, err := f.pool.Exec(ctx, "UPDATE organizations SET archived_at=now() WHERE party_id=$1", f.orgID); err != nil {
		t.Fatal(err)
	}
	if rows, status := read(f.ownerToken); status != 200 || len(rows) != 0 {
		t.Fatalf("archived tenant still listed: status=%d rows=%+v", status, rows)
	}
	if err := f.auth.Logout(ctx, f.memberToken); err != nil {
		t.Fatal(err)
	}
	if status, _ := bearerRead(); status != 401 {
		t.Fatalf("revoked session still exposes workspace list: status=%d", status)
	}
}
