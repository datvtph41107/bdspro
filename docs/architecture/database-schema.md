# Canonical database bootstrap

Scope: fresh databases only. Replaces the 42-step migration history at [2c6f5ba](https://github.com/datvtph41107/bdspro/tree/2c6f5ba5c86e1c46cb7ce3b5b5ad69ef5e932109/db/migration). The old files remain in Git history. Do not use this source with a database created by that ledger. Stop old writers; use a new empty database and explicitly plan data transfer if required. No automatic reset, force-version, drop of existing databases, or in-place upgrade.

## Rules

- One table per numbered up/down pair; up owns its final columns, constraints and indexes, down drops only that table without CASCADE. Foreign keys point backward in order.
- Lowercase snake_case, plural semantic nouns; job_outbox is the established operational pattern name. No broad schema redesign. Keep query-backed storage including email_link_challenges and OAuth server tables pending independent consumer review.
- Existing Go query function names/API contracts remain stable. SQLC models follow physical table names.
- parties carries an explicit bootstrap marker checked by validateSchema; old version numbers cannot be mistaken for new partial bootstrap versions. The existing name/column validation is not a complete database schema diff.
- New migrations after this baseline must be forward-only appended pairs; do not edit already deployed versions.

## Tables in creation order

| Version | Table | Purpose |
| --- | --- | --- |
| 000001 | parties | Shared identity |
| 000002 | persons | Person subtype |
| 000003 | organizations | Organization subtype |
| 000004 | accounts | Login account |
| 000005 | person_profiles | Person profile |
| 000006 | account_settings | Account settings |
| 000007 | organization_members | Organization membership lifecycle |
| 000008 | teams | Team |
| 000009 | team_members | Team membership |
| 000010 | team_invites | Invitation lifecycle and recipient binding |
| 000011 | sessions | Login session |
| 000012 | password_credentials | Password credential |
| 000013 | phone_credentials | Verified phone |
| 000014 | oauth2_credentials | External OAuth/OIDC identity |
| 000015 | email_verification_codes | Email verification codes (existing query contract) |
| 000016 | email_link_challenges | Email link token challenge (existing query contract) |
| 000017 | password_reset_tokens | Password reset token |
| 000018 | verifications | Browser-bound email/phone verification |
| 000019 | google_flows | Google login/link/reauth flow |
| 000020 | auth_proofs | Action-bound recent authentication proof |
| 000021 | contact_link_challenges | Session-bound email/phone link challenge |
| 000022 | account_login_events | Account login event |
| 000023 | audit_events | Audit event |
| 000024 | oauth_clients | OAuth client |
| 000025 | oauth_auth_codes | OAuth authorization code |
| 000026 | oauth_access_tokens | OAuth access token |
| 000027 | oauth_refresh_tokens | OAuth refresh token |
| 000028 | oauth_grants | OAuth grant |
| 000029 | job_outbox | Operational job outbox |

## Changes from the historical final schema

- verify_emails -> email_verification_codes: rows represent issued codes, not verified emails.
- account_connection_histories -> account_login_events: action is constrained to login; preserve query and API behavior.
- contact_links -> contact_link_challenges: temporary session-bound code challenge, not a durable contact link.
- phone_otp_challenges and phone_signup_challenges omitted from fresh bootstrap: previous migration explicitly marked them as legacy with no runtime reader/writer. This does not delete their data in any existing database.
- Fold accounts registration/reset version, organization owner account binding, final audit action/payload checks, team recipient binding, Google auth intent and recent auth proof changes directly into their owning table's CREATE TABLE. Keep all existing checks and indexes; renamed-table indexes use current names. No historical row copy, DROP or ALTER in up files.
- Retain existing separate code/token tables with SQLC queries; a separate requirement can establish which are obsolete. No change to credential security, session lifetimes, membership rules or HTTP API.

## Run

Create a NEW disposable dev database (for example `bdspro_bootstrap`), configure `database.DB_SOURCE` to it, and run `make server` in development or `make migrateup` first. Do not run new code against an old ledger. Use `make generated` with sqlc v1.30.0 to reproduce generated code. Runtime-first verification: start server and call real APIs; deferred automated checks remain opt-in.
