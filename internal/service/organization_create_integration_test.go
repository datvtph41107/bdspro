package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Simulates an account whose referenced person cannot be inserted as an owner.
// The actual workflow, SQLC queries, and PostgreSQL transaction stay real.
type organizationInvalidOwnerStore struct {
	db.Store
}

func (s organizationInvalidOwnerStore) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	return db.Account{ID: id, PersonID: -1}, nil
}

func TestOrganizationCreateWorkflowPostgres(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("set BDS_TEST_DB_URL only for a disposable PostgreSQL database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes without BDS_TEST_MODE=disposable")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing database writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	personID, err := store.CreateParty(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreatePerson(ctx, personID); err != nil {
		t.Fatal(err)
	}
	account, err := store.CreateAccount(ctx, db.CreateAccountParams{
		PersonID: personID,
		Email:    uuid.NewString() + "@example.test",
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := NewOrganizationService(store)
	created, err := svc.Create(ctx, account.ID, "  Workflow owner  ")
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Workflow owner" || created.PartyID <= 0 {
		t.Fatalf("unexpected organization: %+v", created)
	}
	member, err := store.ListOrganizationMembersForOwner(ctx, db.ListOrganizationMembersForOwnerParams{
		OrganizationID: created.PartyID,
		ActorAccountID: account.ID,
	})
	if err != nil || len(member) != 1 || member[0].Role != "owner" || member[0].PersonID != account.PersonID {
		t.Fatalf("workflow must create exactly one owner membership: %+v, %v", member, err)
	}

	failedName := "Rollback " + uuid.NewString()
	invalidOwner := NewOrganizationService(organizationInvalidOwnerStore{Store: store})
	_, err = invalidOwner.Create(ctx, account.ID, failedName)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" ||
		pgErr.ConstraintName != "organization_members_person_id_fkey" {
		t.Fatalf("expected failed owner insert to propagate PostgreSQL FK: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM organizations WHERE name=$1", failedName).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed owner insertion committed organization")
	}
}
