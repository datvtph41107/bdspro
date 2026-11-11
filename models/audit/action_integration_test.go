package audit_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	audit_model "github.com/datvtph41107/bdspro/models/audit"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This wrapper injects a failure after the real SQLC state mutation, inside
// the nested PostgreSQL transaction, without changing production SQLC code.
type rejectingAuditDB struct {
	db.DBTX
}

var errAuditUnavailable = errors.New("injected audit store failure")

func (f rejectingAuditDB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(query, "INSERT INTO audit_events") {
		return pgconn.CommandTag{}, errAuditUnavailable
	}
	return f.DBTX.Exec(ctx, query, args...)
}

type atomicAuditStore struct {
	db.Store
	outer     pgx.Tx
	rejectLog bool
}

func (s *atomicAuditStore) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	inner, err := s.outer.Begin(ctx)
	if err != nil {
		return err
	}
	defer inner.Rollback(ctx)
	var conn db.DBTX = inner
	if s.rejectLog {
		conn = rejectingAuditDB{DBTX: inner}
	}
	if err := fn(db.New(conn)); err != nil {
		return err
	}
	return inner.Commit(ctx)
}

func TestAuditActionsTransactionalAcrossAuthAndTeam(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires migrated disposable PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("audit integration requires disposable mode")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing audit writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	outer, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer outer.Rollback(context.Background())
	q := db.New(outer)
	store := &atomicAuditStore{Store: db.NewStore(pool), outer: outer}
	makeAccount := func() db.Account {
		t.Helper()
		personID, err := db.CreatePersonWithParty(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		a, err := q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: personID, Email: "audit-actions-" + uuid.NewString() + "@example.test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := q.CreateAccountSettings(ctx, a.ID); err != nil {
			t.Fatal(err)
		}
		return a
	}
	owner, recipient := makeAccount(), makeAccount()
	newSession := func() uuid.UUID {
		t.Helper()
		id := uuid.New()
		_, err := q.CreateSession(ctx, db.CreateSessionParams{
			ID: id, AccountID: owner.ID,
			RefreshToken: "sha256:" + strings.Repeat("b", 64),
			UserAgent:    "audit action integration", ClientIp: "203.0.113.8",
			ExpiresAt: time.Now().Add(time.Hour),
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	revoked := newSession()
	if err := auth_model.RevokeOwnedSession(db.WithStore(ctx, store), owner.ID, revoked, revoked); err != nil {
		t.Fatalf("revoke session + audit: %v", err)
	}
	session, err := q.GetSession(ctx, revoked)
	if err != nil || !session.IsBlocked {
		t.Fatalf("session update not committed: %+v %v", session, err)
	}
	// A second session must remain live when the audit insert fails.
	notRevoked := newSession()
	store.rejectLog = true
	if err := auth_model.RevokeOwnedSession(db.WithStore(ctx, store), owner.ID, notRevoked, notRevoked); !errors.Is(err, errAuditUnavailable) {
		t.Fatalf("missing rollback on audit write error: %v", err)
	}
	session, err = q.GetSession(ctx, notRevoked)
	if err != nil || session.IsBlocked {
		t.Fatalf("failed audit erroneously revoked refresh session: %+v %v", session, err)
	}
	store.rejectLog = false

	org, err := organization_model.CreateOrganization(ctx, store, owner.ID, "audit test "+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
		OrganizationID: org.ID, PersonID: recipient.PersonID, Role: "member",
	}); err != nil {
		t.Fatal(err)
	}
	team, err := q.CreateTeamForOwner(ctx, db.CreateTeamForOwnerParams{
		OrganizationID: org.ID, ActorAccountID: owner.ID,
		Name: "audit team", Description: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.AddTeamUserForOwner(ctx, db.AddTeamUserForOwnerParams{
		TeamID: team.ID, ActorAccountID: owner.ID, PersonID: recipient.PersonID,
	}); err != nil {
		t.Fatal(err)
	}
	store.rejectLog = true
	if err := organization_model.RemoveTeamUser(ctx, store, owner.ID, team.ID, recipient.PersonID); !errors.Is(err, errAuditUnavailable) {
		t.Fatalf("failed audit did not cancel member removal: %v", err)
	}
	var members int
	if err := outer.QueryRow(ctx, "SELECT count(*) FROM team_members WHERE team_id=$1 AND person_id=$2", team.ID, recipient.PersonID).Scan(&members); err != nil || members != 1 {
		t.Fatalf("member removal survived failed audit: count=%d err=%v", members, err)
	}
	store.rejectLog = false
	if err := organization_model.RemoveTeamUser(ctx, store, owner.ID, team.ID, recipient.PersonID); err != nil {
		t.Fatalf("member removal + audit: %v", err)
	}
	if err := outer.QueryRow(ctx, "SELECT count(*) FROM team_members WHERE team_id=$1 AND person_id=$2", team.ID, recipient.PersonID).Scan(&members); err != nil || members != 0 {
		t.Fatalf("member removal not committed: count=%d err=%v", members, err)
	}
	// A duplicate delete is not an audited successful action.
	if err := organization_model.RemoveTeamUser(ctx, store, owner.ID, team.ID, recipient.PersonID); !errors.Is(err, organization_model.ErrTeamMemberNotFound) {
		t.Fatalf("duplicate delete not rejected: %v", err)
	}
	// PostgreSQL CHECK constraints must reject NULL actor/scope and keep
	// the old login-proof requirements after the method columns became nullable.
	for _, malformed := range []string{
		"INSERT INTO audit_events(action, scope_type, scope_ref, target_type, target_ref, ip_address) VALUES ('team.member.removed', 'team', '8', 'person', '9', '')",
		"INSERT INTO audit_events(action, actor_account_id, scope_type, target_type, target_ref, ip_address) VALUES ('team.member.removed', 3, 'team', 'person', '9', '')",
		"INSERT INTO audit_events(action, ip_address) VALUES ('auth.login.rejected', '')",
	} {
		if _, err := outer.Exec(ctx, "SAVEPOINT malformed_audit"); err != nil {
			t.Fatal(err)
		}
		if _, err := outer.Exec(ctx, malformed); err == nil {
			t.Fatalf("nullable action payload bypassed audit constraints: %s", malformed)
		}
		if _, err := outer.Exec(ctx, "ROLLBACK TO SAVEPOINT malformed_audit"); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := outer.Query(ctx, `SELECT action, actor_account_id, scope_type, scope_ref, target_type, target_ref,
		method, reason, subject_mac
		FROM audit_events WHERE actor_account_id=$1 ORDER BY id`, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var action, scopeType, scopeRef, targetType, targetRef string
		var actorID int64
		var method, reason *string
		var mac []byte
		if err := rows.Scan(&action, &actorID, &scopeType, &scopeRef, &targetType, &targetRef, &method, &reason, &mac); err != nil {
			t.Fatal(err)
		}
		if actorID != owner.ID || method != nil || reason != nil || mac != nil {
			t.Fatalf("non-login action leaked credentials or wrong actor: %s", action)
		}
		switch action {
		case audit_model.ActionSessionRevoked:
			if scopeType != "account" || scopeRef != fmt.Sprint(owner.ID) ||
				targetType != "session" || targetRef != revoked.String() {
				t.Fatal("incorrect session scope/target")
			}
		case audit_model.ActionTeamMemberRemoved:
			if scopeType != "team" || scopeRef != fmt.Sprint(team.ID) ||
				targetType != "person" || targetRef != fmt.Sprint(recipient.PersonID) {
				t.Fatal("incorrect membership scope/target")
			}
		default:
			t.Fatalf("unexpected audit action: %s", action)
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 {
		t.Fatalf("want exactly two committed audit events, got %v", actions)
	}
}
