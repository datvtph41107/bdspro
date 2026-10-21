package integration

import (
	"context"
	"testing"

	"github.com/datvtph41107/bdspro/auth"
)

func TestDevelopmentSMSOutboxIsDeliveryOnly(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	// Intentionally not part of production migrations. Tests provision this
	// local-only fixture, then tear it down immediately.
	_, err := db.Exec(ctx, `CREATE TABLE dev_sms_outbox (
		id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		phone TEXT NOT NULL,
		otp_code TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '5 minutes')
	)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(context.Background(), "DROP TABLE IF EXISTS dev_sms_outbox")
	})
	sender, err := auth.NewDevelopmentSMSOutbox(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.SendOTP(ctx, "+84901234567", "123456"); err != nil {
		t.Fatal(err)
	}
	var phone, code string
	if err := db.QueryRow(ctx,
		"SELECT phone, otp_code FROM dev_sms_outbox ORDER BY id DESC LIMIT 1").
		Scan(&phone, &code); err != nil {
		t.Fatal(err)
	}
	if phone != "+84901234567" || code != "123456" {
		t.Fatalf("development-only transport returned unexpected destination or code")
	}
	// Auth challenge tables remain independent; they store keyed HMAC, not this row.
	var persisted int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM phone_login_challenges").Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 0 {
		t.Fatalf("delivery adapter improperly created authentication challenge: %d", persisted)
	}
}
