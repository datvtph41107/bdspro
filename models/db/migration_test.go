package db

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	sqlc "github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// This test accepts only an explicitly supplied empty disposable database.
// scripts/test-phone-delivery.py provisions it; normal go test does not touch local data.
func TestAuthStateMigration(t *testing.T) {
	source := os.Getenv("BDSPRO_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("requires disposable BDSPRO_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var existing int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'").Scan(&existing); err != nil {
		t.Fatal(err)
	}
	if existing != 0 {
		t.Fatal("refusing migration test against a nonempty database")
	}
	path, err := filepath.Abs("../../db/migration")
	if err != nil {
		t.Fatal(err)
	}
	migrations := "file://" + path
	if _, err := MigrateUp(ctx, migrations, source, 0); err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	browser := bytes.Repeat([]byte{1}, 32)
	proof := bytes.Repeat([]byte{2}, 32)
	// Coincident IDs/proof digests remain legal across verification kinds.
	for _, item := range []struct{ kind, destination string }{{"email", "test@example.invalid"}, {"phone", "+84912300001"}} {
		_, err := conn.Exec(ctx, "INSERT INTO verifications(kind,id,destination,browser_digest,proof_digest,expires_at,resend_after,failed_attempts) VALUES ($1,$2,$3,$4,$5,now()+interval '15 minutes',now(),2)", item.kind, id, item.destination, browser, proof)
		if err != nil {
			t.Fatal(err)
		}
	}
	var person, account int64
	if err := conn.QueryRow(ctx, "INSERT INTO parties DEFAULT VALUES RETURNING id").Scan(&person); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "INSERT INTO persons(party_id) VALUES ($1)", person); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, "INSERT INTO accounts(person_id) VALUES ($1) RETURNING id", person).Scan(&account); err != nil {
		t.Fatal(err)
	}
	session := uuid.New()
	if _, err := conn.Exec(ctx, "INSERT INTO sessions(id,account_id,refresh_token,user_agent,client_ip,expires_at) VALUES ($1,$2,'test-digest','test','127.0.0.1',now()+interval '1 hour')", session, account); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "INSERT INTO google_flows(browser_digest,phase,subject,remember,expires_at,account_id,session_id,purpose,action) VALUES ($1,'ready','test-subject',false,now()+interval '5 minutes',$2,$3,'reauth','phone.link')", browser, account, session); err != nil {
		t.Fatal(err)
	}
	if err := validateSchema(ctx, conn, latestSchemaVersion); err != nil {
		t.Fatal(err)
	}

	q := sqlc.New(conn)
	email, err := q.LockEmailRegistrationProof(ctx, sqlc.LockEmailRegistrationProofParams{ProofDigest: proof, BrowserDigest: browser})
	if err != nil || email.Email != "test@example.invalid" {
		t.Fatalf("email proof lost: %v", err)
	}
	phone, err := q.LockPhoneVerificationProof(ctx, sqlc.LockPhoneVerificationProofParams{ProofDigest: proof, BrowserDigest: browser})
	if err != nil || phone.PhoneE164 != "+84912300001" {
		t.Fatalf("phone proof lost: %v", err)
	}
	if _, err := q.LockPhoneVerificationProof(ctx, sqlc.LockPhoneVerificationProofParams{ProofDigest: proof, BrowserDigest: bytes.Repeat([]byte{9}, 32)}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("browser binding lost")
	}
	if _, err := q.LockGoogleResult(ctx, browser); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("authenticated flow became anonymous login")
	}
	if _, err := q.LockGoogleAuthenticatedResult(ctx, sqlc.LockGoogleAuthenticatedResultParams{BrowserDigest: browser, AccountID: account, SessionID: uuid.New(), Purpose: "reauth"}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("session binding lost")
	}
	result, err := q.LockGoogleAuthenticatedResult(ctx, sqlc.LockGoogleAuthenticatedResultParams{BrowserDigest: browser, AccountID: account, SessionID: session, Purpose: "reauth"})
	if err != nil || result.Action != "phone.link" {
		t.Fatalf("authenticated flow lost: %v", err)
	}

	if changed, err := MigrateUp(ctx, migrations, source, 0); err != nil || changed {
		t.Fatalf("repeat startup: changed=%v err=%v", changed, err)
	}
	if err := q.DeletePhoneVerification(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := q.LockEmailRegistrationProof(ctx, sqlc.LockEmailRegistrationProofParams{ProofDigest: proof, BrowserDigest: browser}); err != nil {
		t.Fatal("phone deletion crossed verification kind")
	}
}
