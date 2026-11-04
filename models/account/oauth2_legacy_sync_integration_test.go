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

func TestOAuth2LegacyIdentityBridge(t *testing.T) {
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
	account, err := account_model.Register(ctx, store, account_model.RegisterParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "test-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	subject := uuid.NewString()
	_, err = pool.Exec(ctx,
		"INSERT INTO google_identities (google_sub, account_id) VALUES ($1, $2)",
		subject, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	key := auth_model.OAuth2CredentialKey{Issuer: "https://accounts.google.com", Subject: subject}
	id, err := auth_model.GetOAuth2AccountID(ctx, store, key)
	if err != nil || id != account.ID {
		t.Fatalf("legacy insert was not mirrored: id=%d err=%v", id, err)
	}
	other := auth_model.OAuth2CredentialKey{Issuer: key.Issuer, Subject: uuid.NewString()}
	if err := auth_model.LinkOAuth2Credential(ctx, store, account.ID, other); !errors.Is(err, auth_model.ErrOAuth2CredentialAlreadyLinked) {
		t.Fatalf("same account linked twice to one issuer: %v", err)
	}
	_, err = pool.Exec(ctx, "DELETE FROM google_identities WHERE google_sub = $1", subject)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth_model.GetOAuth2AccountID(ctx, store, key); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("deleted legacy identity survived: %v", err)
	}
	// Test the opposite direction for the cut-over and rollback window.
	if err := auth_model.LinkOAuth2Credential(ctx, store, account.ID, other); err != nil {
		t.Fatal(err)
	}
	var gotAccount int64
	if err := pool.QueryRow(ctx,
		"SELECT account_id FROM google_identities WHERE google_sub = $1", other.Subject,
	).Scan(&gotAccount); err != nil || gotAccount != account.ID {
		t.Fatalf("OIDC link not mirrored for legacy reader: %d %v", gotAccount, err)
	}
	if _, err := pool.Exec(ctx,
		"DELETE FROM oauth2_credentials WHERE issuer = $1 AND subject = $2",
		other.Issuer, other.Subject,
	); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT account_id FROM google_identities WHERE google_sub = $1", other.Subject,
	).Scan(&gotAccount); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("legacy identity remained after OIDC unlink: %v", err)
	}
}
