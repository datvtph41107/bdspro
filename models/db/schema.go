package db

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
)

// This is a fresh bootstrap, not an upgrade from the retired 42-step ledger.
const baselineMarker = "BDSPro canonical bootstrap v1: shared party identity."

var canonicalTableOrder = [...]string{
	"parties",
	"persons",
	"organizations",
	"accounts",
	"person_profiles",
	"account_settings",
	"organization_members",
	"teams",
	"team_members",
	"team_invites",
	"sessions",
	"password_credentials",
	"phone_credentials",
	"oauth2_credentials",
	"email_verification_codes",
	"email_link_challenges",
	"password_reset_tokens",
	"verifications",
	"google_flows",
	"auth_proofs",
	"contact_link_challenges",
	"account_login_events",
	"audit_events",
	"oauth_clients",
	"oauth_auth_codes",
	"oauth_access_tokens",
	"oauth_refresh_tokens",
	"oauth_grants",
	"job_outbox",
}

const latestSchemaVersion = uint(len(canonicalTableOrder))

func canonicalTables(version uint) []string {
	if version > latestSchemaVersion {
		return nil
	}
	names := slices.Clone(canonicalTableOrder[:version])
	slices.Sort(names)
	return names
}

func validateSchema(ctx context.Context, conn *pgx.Conn, version uint) error {
	// Version numbers were reset deliberately. Never accept an old partial ledger
	// just because its numeric version and initial table names happen to match.
	var marker string
	if err := conn.QueryRow(ctx, "SELECT coalesce(obj_description(to_regclass('public.parties'), 'pg_class'), '')").Scan(&marker); err != nil {
		return fmt.Errorf("inspect canonical bootstrap marker: %w", err)
	}
	if marker != baselineMarker {
		return fmt.Errorf("database does not belong to the canonical bootstrap; provision a fresh database and migrate data explicitly")
	}
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
		{"accounts", "password_reset_version", 4},
		{"accounts", "username", 4},
		{"team_invites", "recipient_membership_id", 10},
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
