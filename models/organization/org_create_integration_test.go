package organization

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

	created, err := CreateOrganization(ctx, store, account.ID, "  Workflow owner  ")
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Workflow owner" || created.ID <= 0 {
		t.Fatalf("unexpected organization: %+v", created)
	}
	member, err := store.ListOrganizationMembersForOwner(ctx, db.ListOrganizationMembersForOwnerParams{
		OrganizationID: created.ID,
		ActorAccountID: account.ID,
	})
	if err != nil || len(member) != 1 || member[0].Role != "owner" || member[0].PersonID != account.PersonID {
		t.Fatalf("workflow must create exactly one owner membership: %+v, %v", member, err)
	}

	// A missing creator account cannot leave an organization or a Party.
	// This must be checked by the transaction-bound SQLC lookup, not by a
	// mockable preliminary GetAccountByID call on the pool-backed store.
	failedName := "Missing creator " + uuid.NewString()
	_, err = CreateOrganization(ctx, store, -1, failedName)
	if !errors.Is(err, account_model.ErrAccountUnavailable) {
		t.Fatalf("missing account must reject organization creation: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM organizations WHERE name=$1", failedName).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("missing account created organization")
	}
}
