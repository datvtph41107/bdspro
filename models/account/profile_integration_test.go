package account_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestProfileAndSettingsOwnershipInPostgres(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing writes without disposable mode")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := db.NewStore(pool)

	first, err := account_model.Register(ctx, store, account_model.RegisterParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "integration-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := account_model.CreateOAuth2Account(ctx, store, auth_model.OIDCClaims{
		Issuer: "https://issuer.example.test", Subject: uuid.NewString(),
		Email: uuid.NewString() + "@example.test", EmailVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	p1, err := account_model.GetProfile(ctx, store, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := account_model.GetProfile(ctx, store, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p1.PersonID != first.PersonID || p2.PersonID != second.PersonID ||
		!p1.HasPassword || p1.HasOAuth2 || !p2.HasOAuth2 || p2.HasPassword ||
		p1.DisplayName != "" || p1.Locale != "vi" || p2.Currency != "VND" {
		t.Fatalf("incorrect profile/credential projection: first=%+v second=%+v", p1, p2)
	}
	display := "Account One"
	if err := account_model.UpdateProfile(ctx, store, first.ID, account_model.ProfileUpdateOptions{
		DisplayName: &display,
	}); err != nil {
		t.Fatal(err)
	}
	utc := "UTC"
	if err := account_model.UpdateSettings(ctx, store, first.ID, account_model.SettingsUpdateOptions{
		Timezone: &utc,
	}); err != nil {
		t.Fatal(err)
	}
	p1, err = account_model.GetProfile(ctx, store, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	p2, err = account_model.GetProfile(ctx, store, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p1.DisplayName != "Account One" || p1.Timezone != "UTC" || p1.Locale != "vi" ||
		p2.DisplayName != "" || p2.Timezone != "Asia/Ho_Chi_Minh" {
		t.Fatalf("account data crossed owners: first=%+v second=%+v", p1, p2)
	}
	// Independent PATCH fields are updated column-wise in PostgreSQL rather than
	// via a stale read/modify/write copy that can lose a concurrent update.
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	firstName, lastName := "Concurrent First", "Concurrent Last"
	for _, update := range []account_model.ProfileUpdateOptions{
		{FirstName: &firstName},
		{LastName: &lastName},
	} {
		wg.Add(1)
		go func(opts account_model.ProfileUpdateOptions) {
			defer wg.Done()
			errs <- account_model.UpdateProfile(ctx, store, first.ID, opts)
		}(update)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	p1, err = account_model.GetProfile(ctx, store, first.ID)
	if err != nil || p1.FirstName != "Concurrent First" || p1.LastName != "Concurrent Last" {
		t.Fatalf("concurrent partial patches clobbered independent columns: %+v err=%v", p1, err)
	}
}
