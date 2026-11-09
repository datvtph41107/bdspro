# Capability map — start from observable behavior

This is a source map for a new engineer or agent, not a claim that every path is fully verified. Follow the referenced code at the current SHA. Each capability maps from a real user/API action to its owners and tests; the links are checked in CI for existence, not semantic correctness.

## Organization and team

- **HTTP entry:** [routes](../../routers/api/v1/api.go), [org actions](../../routers/api/v1/org/org.go), [team actions](../../routers/api/v1/org/team.go).
- **Request authority:** [API context](../../services/context/api.go), [organization assignment](../../routers/api/v1/organization_context.go).
- **Domain work:** [organization service](../../services/organization/org.go), [team service](../../services/organization/team.go), [organization model](../../models/organization/org.go).
- **SQL owners:** [organization queries](../../db/query/organization.sql), [team queries](../../db/query/team.sql).
- **Evidence:** [transaction integration tests](../../services/organization/org_transaction_integration_test.go), [team concurrency tests](../../models/organization/team_write_concurrency_test.go), [owner integrity](../architecture/g1-owner-account-integrity.md), [doctor detection limits](../architecture/g1-owner-coverage-doctor.md).
- **Do not assume:** an HTTP admission check alone serializes a concurrent owner or membership change; only verified transaction boundaries can establish that guarantee. The ownerless-organization doctor detects certain invalid states but cannot prevent privileged direct SQL mutations.

## First-party authentication and sessions

- **HTTP entry:** [routes](../../routers/api/v1/api.go), [user authentication actions](../../routers/api/v1/user/auth.go).
- **Request authority:** [authentication context](../../services/context/auth.go), [session admission](../../services/context/session.go), [verified Actor model](../../models/auth/actor_context.go).
- **Domain/persistence:** [auth service](../../services/auth/auth.go), [session model](../../models/auth/session.go), [SQLC session queries](../../db/query/session.sql).
- **Evidence:** [live-session contract and limits](../architecture/g1-live-session-authority.md), [HTTP session acceptance](../../routers/g1_live_session_acceptance_test.go), [session concurrency test](../../models/auth/session_revoke_all_race_integration_test.go).
- **Do not assume:** a JWT signature implies a live session, or a live admission check protects a later domain transaction against an intervening revocation.

## Maintain the map

Update a capability when routes, permissions, transaction boundaries or canonical tests change. Do not add speculative packages or duplicate a full call graph in prose. Use the [verify-bdspro skill](../../.cursor/skills/verify-bdspro/SKILL.md) and the [Organization](../../.cursor/skills/verify-bdspro/features/organizations.md) and [Session](../../.cursor/skills/verify-bdspro/features/sessions.md) feature recipes. The current runner exercises in-process Gin with real PostgreSQL/Redis; it does not claim network ingress, provider E2E, or a full app-process launch. A source-map check alone does not prove live behavior.
