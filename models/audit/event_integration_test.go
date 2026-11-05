package audit_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	audit_model "github.com/datvtph41107/bdspro/models/audit"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The PostgreSQL test operates in one rollback-only transaction in a
// disposable database: it proves nullable account ownership, constraints,
// and independent retention without leaving audit data behind.
func TestAuditEventsAnonymousKnownAndRetention(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing authentication event writes outside disposable mode")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing authentication event writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	q := db.New(tx)
	personID, err := db.CreatePersonWithParty(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	account, err := q.CreateAccount(ctx, db.CreateAccountParams{
		PersonID: personID,
		Email:    "failure-" + uuid.NewString() + "@example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	key := strings.Repeat("a", 32)
	proof := audit_model.LoginRejected{
		Method:     audit_model.MethodPassword,
		Reason:     audit_model.ReasonUnknownAccount,
		Identifier: " Ghost@Example.Test ",
		IP:         "203.0.113.1",
	}
	if err := audit_model.RecordLoginRejected(ctx, q, key, proof); err != nil {
		t.Fatal(err)
	}
	proof.AccountID = account.ID
	proof.Reason = audit_model.ReasonInvalidPassword
	proof.Identifier = "ghost@example.test"
	proof.IP = "2001:db8::a"
	if err := audit_model.RecordLoginRejected(ctx, q, key, proof); err != nil {
		t.Fatal(err)
	}
	var anonymousCount, knownCount int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE account_id IS NULL AND method='password' AND ip_address='203.0.113.1'").Scan(&anonymousCount); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE account_id=$1", account.ID).Scan(&knownCount); err != nil {
		t.Fatal(err)
	}
	if anonymousCount != 1 || knownCount != 1 {
		t.Fatalf("wrong account scope: anonymous=%d known=%d", anonymousCount, knownCount)
	}
	rows, err := tx.Query(ctx, "SELECT subject_mac, ip_address, account_id FROM audit_events WHERE ip_address IN ('203.0.113.1','2001:db8::a') ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	var macs [][]byte
	for rows.Next() {
		var mac []byte
		var ip string
		var accountID *int64
		if err := rows.Scan(&mac, &ip, &accountID); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if len(mac) != 32 || bytes.Contains(mac, []byte("ghost@example.test")) {
			rows.Close()
			t.Fatalf("credential event stored plaintext or malformed digest")
		}
		if ip == "203.0.113.1" && accountID != nil {
			rows.Close()
			t.Fatal("unknown identity acquired an account id")
		}
		if ip == "2001:db8::a" && (accountID == nil || *accountID != account.ID) {
			rows.Close()
			t.Fatal("known identity lost account id")
		}
		macs = append(macs, mac)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	if len(macs) != 2 || !bytes.Equal(macs[0], macs[1]) {
		t.Fatalf("same target must correlate by keyed digest: records=%d", len(macs))
	}

	// Reject an inconsistent method/reason pair without poisoning the
	// encompassing rollback-only test transaction.
	if _, err := tx.Exec(ctx, "SAVEPOINT invalid_security_event"); err != nil {
		t.Fatal(err)
	}
	bad := db.InsertAuditEventParams{
		Method: audit_model.MethodPassword, Reason: audit_model.ReasonInvalidOTP,
		SubjectMac: make([]byte, 32), IpAddress: "203.0.113.2",
	}
	if err := q.InsertAuditEvent(ctx, bad); err == nil {
		t.Fatal("invalid method/reason passed PostgreSQL constraint")
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT invalid_security_event"); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now().UTC().Add(-90 * 24 * time.Hour)
	if _, err := tx.Exec(ctx,
		"UPDATE audit_events SET created_at=$1 WHERE account_id IS NULL AND ip_address='203.0.113.1'",
		cutoff.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	deleted, err := q.DeleteOldAuditEvents(ctx, db.DeleteOldAuditEventsParams{Cutoff: cutoff, Take: 1000})
	if err != nil || deleted < 1 {
		t.Fatalf("expired anonymous event not deleted: count=%d err=%v", deleted, err)
	}
	var remaining int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE account_id=$1", account.ID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("retention removed a recent known-account event: %d", remaining)
	}
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE account_id IS NULL AND ip_address='203.0.113.1'").Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("retention left expired anonymous event: %d", remaining)
	}
}
