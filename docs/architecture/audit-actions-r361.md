# R36.1 — Shared audit event model, not one table per action

## Why this migration exists

Migration 000020 created one `audit_events` table but constrained every
record to `auth.login.rejected`, with required `method`, `reason`, and
`subject_mac`. Those are valid fields for rejected credentials, but not
for every security event. Making another `session_events` or
`team_member_events` table would fragment the investigation trail.

R36.1 uses **one migration, 000021**, to evolve the existing table in
place. It preserves all R35.1 login rejection rows, nullable known-account
ownership and HMAC correlation, and adds concrete, bounded actor/scope/target
fields for two **implemented** domain changes:

| Action | Actor | Scope | Target | Persistence |
| --- | --- | --- | --- | --- |
| `auth.login.rejected` | Unknown | Known account if available | HMAC-protected identifier | Best-effort (legacy) |
| `auth.session.revoked` | Authenticated account | Owning account | Session UUID | Required, same transaction as session block |
| `team.member.removed` | Authenticated owner | Team | Removed person ID | Required, same transaction as membership DELETE |

The existing `models/audit.RecordLoginRejected` is unchanged. The new
`models/audit.RecordSessionRevoked` and `RecordTeamMemberRemoved` are
narrow methods that only invoke the canonical SQLC insert. Auth and
organization models own the source of their business decisions and call
audit from the same PostgreSQL transaction. No Gin types, new event
dispatcher, fake interface registry, or per-action service exists.

## Evidence and invariants

* A missing/revoked session or a missing team member does not generate an
  authenticated-success audit event.
* A PostgreSQL audit INSERT error rolls back the session/membership mutation
  in a nested transaction. Tests inject the failure *after* SQL mutation.
* No raw refresh JWT, password, OTP, email or request body is stored.
  Identifiers stored in actor/scope/target are references, not credentials.
* Old rejected-login events remain valid, can still be pruned, and have no
  actor/scope/target columns. Existing success histories are not migrated
  into audit: `account_connection_histories` is a user-facing read model.
* `000021.down.sql` refuses to delete newly recorded actions silently.
  Rollback requires an explicit reviewed evidence-retention decision first.

## Scope limitation

This PR does not invent generalized event metadata or a global event bus.
It does not claim a compliance-grade immutable audit log. Actor account IDs are retained as numeric audit references even if the
account record is deleted. They are deliberately not foreign keys: cascading
delete would erase evidence and SET NULL would violate the authenticated-
action invariant. Retention and privacy deletion policy must explicitly
account for these numeric references. Access-control
policies and paginated, scoped audit reads are not exposed in HTTP; an
authorized investigation use case must be approved first.

The action CHECK and payload CHECK require a **reviewed migration** to add
another action. We prefer the slight migration cost over silently accepting
arbitrary actions or sensitive JSON.

Gitea reference:
* `models/audit/audit_event.go` — one audit-event table, actor/scope/origin
* `services/audit/event.go` — event recording ownership
* `models/audit/action.go` — explicit action names

Laravel DRX reference:
* `user_connection_histories` — per-user connection history distinct from
  broadcast `UserSessionRegistered`, not a generalized audit system.
