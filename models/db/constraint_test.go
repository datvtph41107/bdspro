package db

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Exercise the actual migrated schema, not synthetic PostgreSQL error values.
// All fixture writes remain in an outer transaction that is never committed.
func TestUniqueConstraintContract(t *testing.T) {
	source := os.Getenv("BDSPRO_CONSTRAINT_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("requires disposable migrated BDSPRO_CONSTRAINT_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	id := func(query string, args ...any) int64 {
		t.Helper()
		var value int64
		if err := tx.QueryRow(ctx, query, args...).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	tag := strings.ReplaceAll(uuid.NewString(), "-", "")
	email := tag + "@example.invalid"
	username := "t_" + tag[:24]
	person := id("INSERT INTO parties DEFAULT VALUES RETURNING id")
	exec("INSERT INTO persons(party_id) VALUES ($1)", person)
	account := id("INSERT INTO accounts(person_id,email,username,is_email_verified) VALUES ($1,$2,$3,true) RETURNING id", person, email, username)
	other := id("INSERT INTO parties DEFAULT VALUES RETURNING id")
	exec("INSERT INTO persons(party_id) VALUES ($1)", other)
	otherAccount := id("INSERT INTO accounts(person_id) VALUES ($1) RETURNING id", other)
	org := id("INSERT INTO parties DEFAULT VALUES RETURNING id")
	exec("INSERT INTO organizations(party_id,name) VALUES ($1,'fixture')", org)
	membership := id("INSERT INTO organization_members(organization_id,person_id,role) VALUES ($1,$2,'member') RETURNING id", org, person)
	team := id("INSERT INTO teams(organization_id,name) VALUES ($1,'fixture') RETURNING id", org)
	exec("INSERT INTO team_members(team_id,person_id) VALUES ($1,$2)", team, person)
	digest := sha256.Sum256([]byte(tag))
	invite := "INSERT INTO team_invites(id,team_id,invited_email,invited_by_account_id,token_hash,expires_at,recipient_account_id,recipient_membership_id) VALUES ($1,$2,$3,$4,$5,now()+interval '1 hour',$4,$6)"
	exec(invite, uuid.New(), team, email, account, digest[:], membership)
	secondDigest := sha256.Sum256([]byte(tag + "second"))
	// Use a fixture phone not already present in this disposable database.
	var phone string
	if err := tx.QueryRow(ctx, "SELECT '+849'||lpad(n::text,8,'0') FROM generate_series(0,999) n WHERE NOT EXISTS (SELECT 1 FROM phone_credentials WHERE phone_e164='+849'||lpad(n::text,8,'0')) LIMIT 1").Scan(&phone); err != nil {
		t.Fatal(err)
	}
	exec("INSERT INTO phone_credentials(account_id,phone_e164) VALUES ($1,$2)", account, phone)
	exec("INSERT INTO oauth2_credentials(issuer,subject,account_id) VALUES ('https://accounts.google.com',$1,$2)", tag, account)

	for _, tc := range []struct {
		name       string
		constraint string
		query      string
		args       []any
	}{
		{"email", "accounts_email_login_uq", "UPDATE accounts SET email=$1 WHERE id=$2", []any{strings.ToUpper(email), otherAccount}},
		{"username", "accounts_username_uq", "UPDATE accounts SET username=$1 WHERE id=$2", []any{username, otherAccount}},
		{"phone", "phone_credentials_phone_e164_key", "INSERT INTO phone_credentials(account_id,phone_e164) VALUES ($1,$2)", []any{otherAccount, phone}},
		{"google subject", "oauth2_credentials_pkey", "INSERT INTO oauth2_credentials(issuer,subject,account_id) VALUES ('https://accounts.google.com',$1,$2)", []any{tag, otherAccount}},
		{"google account", "oauth2_credentials_account_issuer_uq", "INSERT INTO oauth2_credentials(issuer,subject,account_id) VALUES ('https://accounts.google.com',$1,$2)", []any{tag + "other", account}},
		{"organization member", "organization_members_one_open_uq", "INSERT INTO organization_members(organization_id,person_id,role) VALUES ($1,$2,'member')", []any{org, person}},
		{"team name", "teams_org_name_uq", "INSERT INTO teams(organization_id,name) VALUES ($1,' FIXTURE ')", []any{org}},
		{"team member", "team_members_pkey", "INSERT INTO team_members(team_id,person_id) VALUES ($1,$2)", []any{team, person}},
		{"open invite", "team_invites_open_uq", invite, []any{uuid.New(), team, email, account, secondDigest[:], membership}},
		{"team primary key", "teams_pkey", "INSERT INTO teams(id,organization_id,name) OVERRIDING SYSTEM VALUE VALUES ($1,$2,'different')", []any{team, org}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			savepoint, err := tx.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer savepoint.Rollback(context.Background())
			_, err = savepoint.Exec(ctx, tc.query, tc.args...)
			if !IsUniqueViolation(err, tc.constraint) {
				t.Fatalf("expected unique constraint %s: %v", tc.constraint, err)
			}
			if IsUniqueViolation(err, tc.constraint+"_other") {
				t.Fatal("unrelated constraint matched")
			}
			if err := savepoint.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, tc := range []struct {
		code  string
		query string
		args  []any
	}{
		{"23503", "INSERT INTO teams(organization_id,name) VALUES (-1,'fixture')", nil},
		{"23514", "INSERT INTO teams(organization_id,name) VALUES ($1,'')", []any{org}},
	} {
		t.Run(tc.code, func(t *testing.T) {
			savepoint, err := tx.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer savepoint.Rollback(context.Background())
			_, err = savepoint.Exec(ctx, tc.query, tc.args...)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != tc.code || IsUniqueViolation(err, pgErr.ConstraintName) {
				t.Fatalf("expected non-unique SQLSTATE %s: %v", tc.code, err)
			}
			if err := savepoint.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM accounts WHERE id=$1 OR id=$2)", account, otherAccount).Scan(&exists); err != nil || exists {
		t.Fatalf("fixture rollback failed: exists=%v err=%v", exists, err)
	}
}
