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
	identity := auth_model.OIDCClaims{
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
	credentials, err := auth_model.ListOAuth2CredentialsForAccount(ctx, store, account.ID)
	if err != nil || len(credentials) != 1 || credentials[0].Issuer != identity.Issuer ||
		credentials[0].Subject != identity.Subject {
		t.Fatalf("unexpected account credential list: %#v %v", credentials, err)
	}
	if err := auth_model.LinkOAuth2Credential(ctx, store, account.ID, firstKey); err != nil {
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
	if err := auth_model.LinkOAuth2Credential(ctx, store, secondAccount.ID, firstKey); !errors.Is(err, auth_model.ErrOAuth2CredentialAlreadyLinked) {
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
	racing := auth_model.OIDCClaims{
		Issuer: "https://oidc-race.example.test", Subject: uuid.NewString(),
		Email: uuid.NewString() + "@example.test", EmailVerified: true,
	}
	type outcome struct {
		accountID int64
		err       error
	}
	results := make(chan outcome, 2)
	for range 2 {
		go func() {
			created, createErr := account_model.CreateOAuth2Account(ctx, store, racing)
			results <- outcome{accountID: created.ID, err: createErr}
		}()
	}
	successes := 0
	var winner int64
	for range 2 {
		result := <-results
		if result.err == nil {
			successes++
			winner = result.accountID
		}
	}
	if successes != 1 || winner <= 0 {
		t.Fatalf("concurrent identical external identities created %d accounts", successes)
	}
	var count int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM accounts WHERE email = $1", racing.Email,
	).Scan(&count); err != nil || count != 1 {
		t.Fatalf("concurrent account creation persisted %d rows: %v", count, err)
	}
	raceKey, err := racing.Key()
	if err != nil {
		t.Fatal(err)
	}
	linked, err := auth_model.GetOAuth2AccountID(ctx, store, raceKey)
	if err != nil || linked != winner {
		t.Fatalf("concurrent registration lost identity binding: got=%d want=%d err=%v", linked, winner, err)
	}
}
