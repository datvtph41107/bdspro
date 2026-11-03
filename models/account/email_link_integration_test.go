package account_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestEmailLinkModelAtomicity uses real PostgreSQL and SQLC. A failed proof
// must not attach any email, and the successful proof must be single-use.
func TestEmailLinkModelAtomicity(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable bdspro_test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes outside disposable mode")
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

	partyID, err := store.CreateParty(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreatePerson(ctx, partyID); err != nil {
		t.Fatal(err)
	}
	phoneOnly, err := store.CreatePhoneOnlyAccount(ctx, partyID)
	if err != nil {
		t.Fatal(err)
	}
	alreadyTaken, err := account_model.Register(ctx, store, account_model.RegisterParams{
		Email:        uuid.NewString() + "@example.test",
		PasswordHash: "test-only",
	})
	if err != nil {
		t.Fatal(err)
	}
	token := sha256.Sum256([]byte(uuid.NewString()))
	if err := account_model.BeginEmailLink(ctx, store, phoneOnly.ID, alreadyTaken.Email, token[:]); !errors.Is(err, account_model.ErrEmailLinkTaken) {
		t.Fatalf("existing email must not be attached: %v", err)
	}
	newEmail := uuid.NewString() + "@example.test"
	if err := account_model.BeginEmailLink(ctx, store, phoneOnly.ID, newEmail, token[:]); err != nil {
		t.Fatal(err)
	}
	if err := account_model.BeginEmailLink(ctx, store, phoneOnly.ID, newEmail, token[:]); !errors.Is(err, account_model.ErrEmailLinkCooldown) {
		t.Fatalf("resend cooldown not enforced: %v", err)
	}
	wrong := sha256.Sum256([]byte("incorrect-email-proof"))
	if err := account_model.ConfirmEmailLink(ctx, store, phoneOnly.ID, wrong[:]); !errors.Is(err, account_model.ErrEmailLinkInvalid) {
		t.Fatalf("incorrect email proof must fail: %v", err)
	}
	if err := account_model.ConfirmEmailLink(ctx, store, phoneOnly.ID, token[:]); err != nil {
		t.Fatal(err)
	}
	updated, err := store.GetAccountByID(ctx, phoneOnly.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != phoneOnly.ID || updated.Email != newEmail || !updated.IsEmailVerified {
		t.Fatalf("link must update same account with verified email: %+v", updated)
	}
	if err := account_model.ConfirmEmailLink(ctx, store, phoneOnly.ID, token[:]); !errors.Is(err, account_model.ErrEmailLinkInvalid) {
		t.Fatalf("consumed proof replay must fail: %v", err)
	}
}
