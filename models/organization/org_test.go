package organization_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	organization "github.com/datvtph41107/bdspro/models/organization"
	apperrors "github.com/datvtph41107/bdspro/modules/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests require a dedicated migrated disposable database; skipping them
// must be detected by the integration CI acceptance gate, not mistaken for
// PostgreSQL correctness.
func integrationStore(t *testing.T) (context.Context, db.Store) {
	t.Helper()
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires BDS_TEST_DB_URL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("BDS_TEST_MODE=disposable is mandatory")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing to write outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, db.NewStore(pool)
}

func makeActor(t *testing.T, ctx context.Context, store db.Store) db.Account {
	t.Helper()
	id, err := store.CreateParty(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreatePerson(ctx, id); err != nil {
		t.Fatal(err)
	}
	a, err := store.CreateAccount(ctx, db.CreateAccountParams{PersonID: id, Email: uuid.NewString() + "@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestCreateOrganizationAndOwnerPostgres(t *testing.T) {
	ctx, store := integrationStore(t)
	owner := makeActor(t, ctx, store)
	outsider := makeActor(t, ctx, store)
	name := "Gitea-style " + uuid.NewString()
	created, err := organization.CreateOrganization(ctx, store, owner.ID, "  "+name+"  ")
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != name || created.PartyID <= 0 {
		t.Fatalf("created %+v", created)
	}
	members, err := organization.GetOrgMembers(ctx, store, owner.ID, created.PartyID)
	if err != nil || len(members) != 1 || members[0].PersonID != owner.PersonID || members[0].Role != "owner" {
		t.Fatalf("owner membership %+v err=%v", members, err)
	}

	_, err = organization.UpdateOrganization(ctx, store, outsider.ID, created.PartyID, "tampered")
	if !errors.Is(err, apperrors.OrganizationOwnerRequired) {
		t.Fatalf("outsider update: %v", err)
	}
	visible, err := organization.GetForAccount(ctx, store, owner.ID, created.PartyID)
	if err != nil || visible.Organization.Name != name {
		t.Fatalf("unauthorized mutation persisted: %+v err=%v", visible, err)
	}
}
