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

func TestOAuth2ArchiveDoesNotReceiveNewCredentials(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing non-disposable database")
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

	subject := uuid.NewString()
	account, err := account_model.CreateOAuth2Account(ctx, db.NewStore(pool), auth_model.OIDCClaims{
		Issuer: "https://accounts.google.com", Subject: subject,
		Email: uuid.NewString() + "@example.test", EmailVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var archived int64
	err = pool.QueryRow(ctx,
		"SELECT account_id FROM oauth2_credentials_archive WHERE subject = $1", subject,
	).Scan(&archived)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("new OAuth2 credentials unexpectedly wrote to legacy archive: %v", err)
	}
	var linked int64
	err = pool.QueryRow(ctx,
		"SELECT account_id FROM oauth2_credentials WHERE issuer=$1 AND subject=$2",
		"https://accounts.google.com", subject,
	).Scan(&linked)
	if err != nil || linked != account.ID {
		t.Fatalf("active credential was lost: linked=%d err=%v", linked, err)
	}
}
