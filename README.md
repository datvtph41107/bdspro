# BDSPro — Canonical Core 01–05

Implementation branch: **dev**. Canonical V0.4's 164 tables remain a logical design candidate; this repository implements the first bounded, working slice from Account to Organization to Source Intake.

## What is implemented

| Migration | Write-core responsibilities |
| --- | --- |
| 000001_party_identity | Party, Person, Organization, PersonProfile; deferred C01 exactly-one subtype at commit |
| 000002_account_authentication | Account, emails/phones, password, passkeys, external identities, service principals, email verification, Sessions |
| 000003_organization_authorization | Permissions, scoped Roles, Membership, Blocks, Invitation, RoleAssignment, primary workspace administrator |
| 000004_organization_onboarding | New business registration, existing business Claim, platform Reviewer authority, verified registry identifiers |
| 000005_source_intake | Unverified source observations, append-only price statements, tenant-safe FK, idempotency |

Runtime: Go 1.27, PostgreSQL 18, pgx/v5, sqlc. HTTP server implements email/password registration + verification, login/session/logout, Organization registration/claim/review/invitation/members, and Source capture/price history.

**Explicit boundary:** Source observation is NOT canonical Property, Listing, legal title, approved publication or proof of current asking price. An Organization claim or declared registration code is NOT permission to administer a real company. Workspace admin is NOT a legal representative.

## Development start (fresh disposable DB, not existing legacy bdspro)

Run in ~/projects/bdspro after safely synchronizing dev:

1. Ensure PostgreSQL 18 container is running and create a **new** database bdspro_core_dev. Do not migrate against an old database carrying removed Listing migration history.
2. Apply the five ordered migrations in migrations/ using the normal golang-migrate tool and ONE migration version history.
3. Run sqlc generate -f sqlc.source.yaml and go mod tidy (in that order) to produce db/sourcesqlc.
4. Run the server with:
   
       export DATABASE_URL='postgresql://root:secret@localhost:5432/bdspro_core_dev?sslmode=disable'
       export APP_ENV=development
       export BIND_HOST=127.0.0.1
       export PORT=8080
       go run .

5. Register: POST /auth/register with email and password (minimum 12 characters). In **loopback development only**, response includes a verification_token. POST /auth/verify-email with that token, POST /auth/login, retain the session Cookie, then GET /auth/me.
6. Use a *separately provisioned platform reviewer* to approve requests; role setup is intentionally **not** a public unauthenticated "make me admin" endpoint. See docs/implementation/CORE-01-TO-05.md and guarded fixture in CI tests.
7. Organization admins may invite verified Account emails to a scoped Role; accepting invitation creates Membership + RoleAssignment in a transaction.
8. Organization member with sources.create/sources.price_report records an unverified source and an optional reported price. Idempotency-Key required for both capture and independent price report.

### HTTP routes

- POST /auth/register, POST /auth/verify-email, POST /auth/login, GET /auth/me, POST /auth/logout
- POST /organizations/registrations
- POST /organizations/{id}/claims
- POST /platform/organizations/registrations/{id}/approve
- POST /platform/organizations/claims/{id}/approve
- POST /organizations/{id}/invitations
- POST /organizations/invitations/{id}/accept
- GET /organizations/{id}/members
- POST /organizations/{id}/sources
- GET /organizations/{id}/sources/{sourceID}
- POST /organizations/{id}/sources/{sourceID}/price-reports
- GET /organizations/{id}/sources/{sourceID}/price-reports
- GET /health, GET /ready

### Production boundaries

APP_ENV=development only permitted with loopback BIND_HOST; verification token is returned to development clients only. Outside development, SMTP_ADDR and ALLOWED_ORIGIN are required; SMTP mailer enforces STARTTLS. Set HTTPS at reverse proxy for Secure HttpOnly SameSite cookies.

Never expose DB root credentials or an admin-reviewer provisioning endpoint. The current local-development flow is **not a production security release** without mail retry/outbox, account recovery, independent evidence verification, CSRF/abuse controls, role revocation concurrency tests, audit retention, data protection and operational load review.

## Evidence, decisions, safety

- Design, invariant, tradeoffs, status and acceptance: docs/implementation/CORE-01-TO-05.md.
- Canonical candidate: docs/database/final/BDSPro-CANONICAL-DATABASE.dbml.
- Prior causal business context: docs/business/BUSINESS-ATLAS-V1-2026-10-10.md.
- Source module: source/, db/source/query/source.sql, sqlc.source.yaml.
- CI: .github/workflows/bdspro-core.yml; integration_test.go + source/service_test.go.
- Do not reset, clean, force-pull or overwrite local uncommitted work. Local code/DB may differ from GitHub dev; compare before switching branches.
