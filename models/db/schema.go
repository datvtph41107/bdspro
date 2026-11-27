package db

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
)

const latestSchemaVersion = 42

// tableHistory describes persisted names, not service workflow steps.
// Versions <= 41 remain accepted so existing databases can be upgraded in place.
var tableHistory = []struct {
	name  string
	since uint
}{
	{"parties", 1},
	{"persons", 2},
	{"accounts", 3},
	{"sessions", 4},
	{"verify_emails", 5},
	{"organizations", 6},
	{"organization_members", 7},
	{"phone_credentials", 8},
	{"phone_otp_challenges", 9},
	{"job_outbox", 10},
	{"password_credentials", 11},
	{"phone_signup_challenges", 12},
	{"password_reset_tokens", 13},
	{"email_link_challenges", 14},
	{"teams", 15},
	{"team_members", 16},
	{"team_invites", 17},
	{"oauth2_credentials", 18},
	{"person_profiles", 19},
	{"account_settings", 20},
	{"account_connection_histories", 21},
	{"audit_events", 22},
	{"oauth_clients", 23},
	{"oauth_auth_codes", 24},
	{"oauth_access_tokens", 25},
	{"oauth_refresh_tokens", 26},
	{"oauth_grants", 27},
	{"email_registration_challenges", 33},
	{"phone_continuation_challenges", 35},
	{"google_auth_flows", 36},
	{"recent_auth_proofs", 37},
	{"credential_link_challenges", 39},
	{"google_authenticated_intents", 40},
}

func canonicalTables(version uint) []string {
	var names []string
	for _, table := range tableHistory {
		if table.since > version {
			continue
		}
		name := table.name
		if version >= 42 {
			switch name {
			case "email_registration_challenges", "google_authenticated_intents":
				continue
			case "phone_continuation_challenges":
				name = "verifications"
			case "google_auth_flows":
				name = "google_flows"
			case "recent_auth_proofs":
				name = "auth_proofs"
			case "credential_link_challenges":
				name = "contact_links"
			}
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func validateSchema(ctx context.Context, conn *pgx.Conn, version uint) error {
	rows, err := conn.Query(ctx, `SELECT table_name FROM information_schema.tables
        WHERE table_schema='public' AND table_type='BASE TABLE' AND table_name<>'schema_migrations'
        ORDER BY table_name`)
	if err != nil {
		return fmt.Errorf("inspect canonical tables: %w", err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("read canonical tables: %w", err)
	}
	expected := canonicalTables(version)
	if !slices.Equal(names, expected) {
		var missing, extra []string
		for _, name := range expected {
			if !slices.Contains(names, name) {
				missing = append(missing, name)
			}
		}
		for _, name := range names {
			if !slices.Contains(expected, name) {
				extra = append(extra, name)
			}
		}
		return fmt.Errorf("database at migration %d has unexpected schema: missing=%v extra=%v; use an explicit data migration", version, missing, extra)
	}
	for _, column := range []struct {
		table, name string
		since       uint
	}{
		{"accounts", "password_reset_version", 28},
		{"accounts", "username", 33},
		{"team_invites", "recipient_membership_id", 41},
	} {
		var exists bool
		err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns
            WHERE table_schema='public' AND table_name=$1 AND column_name=$2)`, column.table, column.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("inspect canonical columns: %w", err)
		}
		if exists != (version >= column.since) {
			return fmt.Errorf("database at migration %d has inconsistent column %s.%s", version, column.table, column.name)
		}
	}
	return nil
}
