package organization_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// teamWriteConn executes the tested SQLC write on a known PostgreSQL backend.
// This makes pg_blocking_pids evidence of a lock, not a timing assumption.
type teamWriteConn struct {
	db.Store
	conn *pgxpool.Conn
}

func (s teamWriteConn) CreateTeamForOwner(ctx context.Context, p db.CreateTeamForOwnerParams) (db.Team, error) {
	return db.New(s.conn).CreateTeamForOwner(ctx, p)
}

func (s teamWriteConn) RenameTeamForOwner(ctx context.Context, p db.RenameTeamForOwnerParams) (db.Team, error) {
	return db.New(s.conn).RenameTeamForOwner(ctx, p)
}

func (s teamWriteConn) AddTeamUserForOwner(ctx context.Context, p db.AddTeamUserForOwnerParams) (db.TeamMember, error) {
	return db.New(s.conn).AddTeamUserForOwner(ctx, p)
}

func (s teamWriteConn) RemoveTeamUserForOwner(ctx context.Context, p db.RemoveTeamUserForOwnerParams) (int64, error) {
	return db.New(s.conn).RemoveTeamUserForOwner(ctx, p)
}

func (s teamWriteConn) RevokeTeamInviteForOwner(ctx context.Context, p db.RevokeTeamInviteForOwnerParams) (int64, error) {
	return db.New(s.conn).RevokeTeamInviteForOwner(ctx, p)
}

// Invitation creation/acceptance uses db.WithTxResult. Pin its entire
// transaction, not merely one query, to the controlled backend.
func (s teamWriteConn) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(db.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func TestTeamWriteMembershipRevocationRace(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires migrated disposable PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing writes without BDS_TEST_MODE=disposable")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing writes outside bdspro_test")
	}
	cfg.MaxConns = max(cfg.MaxConns, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	store := db.NewStore(pool)

	cases := []struct {
		name            string
		revokeRecipient bool
		expected        error
	}{
		{"create_team", false, organization.ErrTeamOwnerRequired},
		{"rename_team", false, organization.ErrTeamOwnerRequired},
		{"add_team_user_owner", false, organization.ErrTeamOwnerRequired},
		{"add_team_user_recipient", true, organization.ErrTeamOwnerRequired},
		{"remove_team_user", false, organization.ErrTeamMemberNotFound},
		{"create_invite_owner", false, organization.ErrTeamOwnerRequired},
		{"create_invite_recipient", true, organization.ErrTeamOwnerRequired},
		{"revoke_invite_owner", false, organization.ErrTeamInviteInvalid},
		{"accept_invite_recipient", true, organization.ErrTeamInviteInvalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner := makeActor(t, ctx, store)
			recipient := makeActor(t, ctx, store)
			org, err := organization.CreateOrganization(ctx, store, owner.ID, "lock test "+uuid.NewString())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.PartyID, recipient.PersonID); err != nil {
				t.Fatal(err)
			}
			if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
				ID: recipient.ID, Email: recipient.Email,
			}); err != nil {
				t.Fatal(err)
			}
			team, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
				OrganizationID: org.PartyID, ActorAccountID: owner.ID, Name: "Initial",
			})
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "remove_team_user" {
				if _, err := organization.AddTeamUser(ctx, store, owner.ID, team.ID, recipient.PersonID); err != nil {
					t.Fatal(err)
				}
			}
			var proof organization.CreatedTeamInvite
			if tc.name == "revoke_invite_owner" || tc.name == "accept_invite_recipient" {
				proof, err = organization.CreateTeamInvite(ctx, store, owner.ID, team.ID, recipient.Email, time.Hour, "01234567890123456789012345678901")
				if err != nil {
					t.Fatal(err)
				}
			}

			revoker, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer revoker.Release()
			writer, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Release()
			revokerPID := int32(revoker.Conn().PgConn().PID())
			writerPID := int32(writer.Conn().PgConn().PID())
			targetID := owner.PersonID
			if tc.revokeRecipient {
				targetID = recipient.PersonID
			}
			tx, err := revoker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			var ownerMembership int64
			if err := tx.QueryRow(ctx,
				"SELECT id FROM organization_members WHERE organization_id=$1 AND person_id=$2 AND ended_at IS NULL",
				org.PartyID, targetID,
			).Scan(&ownerMembership); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx,
				"UPDATE organization_members SET ended_at=now(), owner_person_id=NULL WHERE id=$1",
				ownerMembership,
			); err != nil {
				t.Fatal(err)
			}

			pinned := teamWriteConn{Store: store, conn: writer}
			result := make(chan error, 1)
			go func() {
				var err error
				switch tc.name {
				case "create_team":
					_, err = organization.CreateTeam(ctx, pinned, organization.CreateTeamOptions{
						OrganizationID: org.PartyID, ActorAccountID: owner.ID, Name: "Unauthorized",
					})
				case "rename_team":
					_, err = organization.RenameTeam(ctx, pinned, owner.ID, team.ID, "Unauthorized")
				case "add_team_user_owner", "add_team_user_recipient":
					_, err = organization.AddTeamUser(ctx, pinned, owner.ID, team.ID, recipient.PersonID)
				case "remove_team_user":
					err = organization.RemoveTeamUser(ctx, pinned, owner.ID, team.ID, recipient.PersonID)
				case "create_invite_owner", "create_invite_recipient":
					_, err = organization.CreateTeamInvite(ctx, pinned, owner.ID, team.ID, recipient.Email,
						time.Hour, "01234567890123456789012345678901")
				case "revoke_invite_owner":
					err = organization.RevokeTeamInvite(ctx, pinned, owner.ID, team.ID, proof.Invite.ID)
				case "accept_invite_recipient":
					_, err = organization.AcceptTeamInvite(ctx, pinned, recipient.ID, proof.Token)
				}
				result <- err
			}()

			// Actual lock evidence is required. A sleep or assumed ordering
			// could hide an authorization bypass under concurrency.
			ticker := time.NewTicker(20 * time.Millisecond)
			defer ticker.Stop()
			timeout := time.NewTimer(5 * time.Second)
			defer timeout.Stop()
			blocked := false
			for !blocked {
				select {
				case err := <-result:
					t.Fatalf("write completed before membership revocation: %v", err)
				case <-ticker.C:
					if err := pool.QueryRow(ctx,
						"SELECT $1::integer = ANY(pg_blocking_pids($2::integer))",
						revokerPID, writerPID,
					).Scan(&blocked); err != nil {
						t.Fatal(err)
					}
				case <-timeout.C:
					t.Fatalf("write did not lock the revoking membership in %s", tc.name)
				}
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-result:
				if !errors.Is(err, tc.expected) {
					t.Fatalf("after revocation expected %v, got %v", tc.expected, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("write did not finish after membership revocation")
			}
			var actualName string
			if err := pool.QueryRow(ctx, "SELECT name FROM teams WHERE id=$1", team.ID).Scan(&actualName); err != nil {
				t.Fatal(err)
			}
			if actualName != "Initial" {
				t.Fatalf("revoked writer renamed team to %q", actualName)
			}
			var added int
			if err := pool.QueryRow(ctx,
				"SELECT count(*) FROM team_members WHERE team_id=$1 AND person_id=$2",
				team.ID, recipient.PersonID,
			).Scan(&added); err != nil {
				t.Fatal(err)
			}
			if tc.name == "remove_team_user" {
				if added != 1 {
					t.Fatal("unauthorized removal deleted the team member")
				}
			} else if added != 0 {
				t.Fatal("unauthorized write created a team member")
			}

			if tc.name == "accept_invite_recipient" {
				// Consumption and membership insertion are one transaction.
				// Failed recipient authorization must not burn the proof.
				var consumed bool
				if err := pool.QueryRow(ctx,
					"SELECT accepted_at IS NOT NULL FROM team_invites WHERE id=$1", proof.Invite.ID,
				).Scan(&consumed); err != nil {
					t.Fatal(err)
				}
				if consumed {
					t.Fatal("failed acceptance consumed one-time invitation")
				}
				if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.PartyID, recipient.PersonID); err != nil {
					t.Fatal(err)
				}
				// A rejoin must never restore a proof from the previous
				// organization membership generation.
				if _, err := organization.AcceptTeamInvite(ctx, store, recipient.ID, proof.Token); !errors.Is(err, organization.ErrTeamInviteInvalid) {
					t.Fatalf("old invitation proof reactivated after rejoin: %v", err)
				}
				fresh, err := organization.CreateTeamInvite(ctx, store, owner.ID, team.ID,
					recipient.Email, time.Hour, "01234567890123456789012345678901")
				if err != nil {
					t.Fatalf("old open proof blocked a new invitation: %v", err)
				}
				if _, err := organization.AcceptTeamInvite(ctx, store, recipient.ID, fresh.Token); err != nil {
					t.Fatalf("valid new-generation proof rejected: %v", err)
				}
			}
		})
	}
}
