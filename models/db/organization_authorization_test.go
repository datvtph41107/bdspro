package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs only on a disposable database with migrations already applied.
// Never pass an application database, even when exercising this test locally.
func TestOrganizationOwnerAuthorizationSQL(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("set BDS_TEST_DB_URL only for an isolated integration database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes without BDS_TEST_MODE=disposable")
	}
	settings, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if settings.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing database writes outside the dedicated bdspro_test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, settings)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)

	createAccount := func() Account {
		t.Helper()
		id, err := store.CreateParty(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.CreatePerson(ctx, id); err != nil {
			t.Fatal(err)
		}
		account, err := store.CreateAccount(ctx, CreateAccountParams{
			PersonID: id,
			Email:    uuid.NewString() + "@example.test",
		})
		if err != nil {
			t.Fatal(err)
		}
		return account
	}
	owner := createAccount()
	outsider := createAccount()
	member := createAccount()

	// SQL authorization fixtures are setup data, not a second production
	// implementation of the create-organization workflow.
	partyID, err := store.CreateParty(ctx)
	if err != nil {
		t.Fatal(err)
	}
	org, err := store.CreateOrganization(ctx, CreateOrganizationParams{
		PartyID: partyID, Name: "Owned by first account",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganizationMember(ctx, CreateOrganizationMemberParams{
		OrganizationID: partyID, PersonID: owner.PersonID, Role: "owner",
	}); err != nil {
		t.Fatal(err)
	}

	denied, err := store.UpdateOrganization(ctx, UpdateOrganizationParams{
		OrganizationID: org.PartyID, ActorAccountID: outsider.ID,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("non-owner update: expected no rows; got %+v %v", denied, err)
	}
	fromOwner, err := store.ListOrganizationsForAccount(ctx, owner.ID)
	if err != nil || len(fromOwner) != 1 || fromOwner[0].Name != org.Name {
		t.Fatalf("unauthorized update changed resource: %+v %v", fromOwner, err)
	}

	_, err = store.AddOrganizationMember(ctx, AddOrganizationMemberParams{
		OrganizationID: org.PartyID, PersonID: member.PersonID, ActorAccountID: outsider.ID,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("outsider added member: %v", err)
	}
	unseen, err := store.ListOrganizationsForAccount(ctx, member.ID)
	if err != nil || len(unseen) != 0 {
		t.Fatalf("denied membership mutated data: %+v %v", unseen, err)
	}

	added, err := store.AddOrganizationMember(ctx, AddOrganizationMemberParams{
		OrganizationID: org.PartyID, PersonID: member.PersonID, ActorAccountID: owner.ID,
	})
	if err != nil || added.Role != "member" {
		t.Fatalf("owner could not add member: %+v %v", added, err)
	}
	_, err = store.AddOrganizationMember(ctx, AddOrganizationMemberParams{
		OrganizationID: org.PartyID, PersonID: member.PersonID, ActorAccountID: owner.ID,
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "organization_members_one_open_uq" {
		t.Fatalf("duplicate active membership must be rejected by unique constraint: %v", err)
	}
	// A failed third insert must not commit the named Organization.
	// Use a unique fixture instead of a global Party count; other package
	// tests can write to the same disposable database concurrently.
	rollbackName := "Rollback probe " + uuid.NewString()
	err = store.ExecTx(ctx, func(q *Queries) error {
		id, err := q.CreateParty(ctx)
		if err != nil {
			return err
		}
		if _, err := q.CreateOrganization(ctx, CreateOrganizationParams{
			PartyID: id, Name: rollbackName,
		}); err != nil {
			return err
		}
		_, err = q.CreateOrganizationMember(ctx, CreateOrganizationMemberParams{
			OrganizationID: id, PersonID: -1, Role: "owner",
		})
		return err
	})
	var ownerErr *pgconn.PgError
	if !errors.As(err, &ownerErr) || ownerErr.Code != "23503" ||
		ownerErr.ConstraintName != "organization_members_person_id_fkey" {
		t.Fatalf("expected person foreign-key constraint, got %v", err)
	}
	var committed int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM organizations WHERE name=$1", rollbackName).Scan(&committed); err != nil {
		t.Fatal(err)
	}
	if committed != 0 {
		t.Fatalf("failed transaction committed %d organization rows", committed)
	}

	// Registration and the Redis delivery intent must commit in one database transaction.
	registered, err := store.RegisterTx(ctx, RegisterTxParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "test-only",
	})
	if err != nil {
		t.Fatal(err)
	}
	var pending int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM job_outbox WHERE (payload->>'account_id')::bigint = $1 AND queued_at IS NULL", registered.ID).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if pending != 1 {
		t.Fatalf("expected one durable verification intent, got %d", pending)
	}

}
