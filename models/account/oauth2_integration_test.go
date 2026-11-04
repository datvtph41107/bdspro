package account_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOAuth2IdentityIsIssuerScopedAndPasswordless(t *testing.T) {
	url := os.Getenv("BDS_TEST_DB_URL")
	if url == "" {
		t.Skip("requires disposable test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes outside disposable mode")
	}
	cfg, err := pgxpool.ParseConfig(url)
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
	store := db.NewStore(pool)
	subject := uuid.NewString()
	identity := auth_model.OIDCIdentity{
		Issuer: "https://oidc-first.example.test", Subject: subject,
		Email: uuid.NewString() + "@example.test", EmailVerified: true,
	}
	account, err := account_model.CreateOAuth2Account(ctx, store, identity)
	if err != nil {
		t.Fatal(err)
	}
	if !account.IsEmailVerified {
		t.Fatal("trusted verified email was not recorded")
	}
	if _, err := store.GetPasswordHash(ctx, account.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("OIDC-only account unexpectedly has a password: %v", err)
	}
	firstKey, err := identity.Key()
	if err != nil {
		t.Fatal(err)
	}
	id, err := auth_model.GetOAuth2AccountID(ctx, store, firstKey)
	if err != nil || id != account.ID {
		t.Fatalf("identity was not linked to created account: %d %v", id, err)
	}
	if err := auth_model.LinkOAuth2Identity(ctx, store, account.ID, firstKey); err != nil {
		t.Fatalf("same account link should be idempotent: %v", err)
	}
	second := identity
	second.Issuer = "https://oidc-second.example.test"
	second.Email = uuid.NewString() + "@example.test"
	secondAccount, err := account_model.CreateOAuth2Account(ctx, store, second)
	if err != nil {
		t.Fatal(err)
	}
	if secondAccount.ID == account.ID {
		t.Fatal("different issuers collapsed to one identity")
	}
	if err := auth_model.LinkOAuth2Identity(ctx, store, secondAccount.ID, firstKey); !errors.Is(err, auth_model.ErrOAuth2IdentityAlreadyLinked) {
		t.Fatalf("foreign account stole identity: %v", err)
	}
	conflict := identity
	conflict.Subject = uuid.NewString()
	_, err = account_model.CreateOAuth2Account(ctx, store, conflict)
	if err == nil {
		t.Fatal("same email silently linked to a new external identity")
	}
	conflictKey, err := conflict.Key()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth_model.GetOAuth2AccountID(ctx, store, conflictKey); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("failed account transaction left external identity: %v", err)
	}
}
