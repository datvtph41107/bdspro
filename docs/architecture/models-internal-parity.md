> **Historical migration snapshot (R17–R19).** This is not the current
> source inventory. For active OAuth2 and model ownership see
> [R26](r26-oauth2-runtime-consolidation.md) and
> [services](services.md). Earlier Google-specific paths and outstanding
> tasks below intentionally describe their state **at the original baseline**.

# Legacy internal → Gitea-style models/modules acceptance

Evidence baseline: `e43ebbd9271f14a94e6ea6f27bb02aa7d618df8f`
(the pre-migration `internal/` tree), compared with the stacked R17–R19
branches. This matrix distinguishes **moved behavior** from **unimplemented
business domains**. A renamed directory alone is not acceptance.

| Former owner | New owner | Verified scope / remaining concern |
| --- | --- | --- |
| `internal/database/store.go` + SQLC files | `models/db/store.go`, `models/db/sqlc/*.gen.go` | Single Store facade; generated source produced by `make generated` |
| `internal/database/tx_create_google_account.go` | `models/account/google.go` | Atomic Google-first Account creation preserved; identity lookup still has `models/db/google_identity.go` |
| `internal/database/tx_create_organization.go` | `models/organization/org.go` | Atomic Party+Organization+owner creation |
| `internal/database/tx_email_link.go` | `models/account/email_link.go` | One-time proof and email ownership |
| `internal/database/tx_password_reset.go` | `models/auth/password.go` | Consume reset token, update credential, revoke sessions in one transaction |
| `internal/database/tx_verify_email*.go` | `models/account/verify_email.go`, `verification_issue.go` | One-time verification, outbox intent handling |
| `internal/database/phone_otp.go` | `models/db/phone_otp.go` | **Outstanding:** phone challenge / failed-attempt business rules still live in database infrastructure |
| `internal/database/tx_phone_signup.go` | `models/db/tx_phone_signup.go` | **Outstanding:** phone-first identity creation still owned by DB package |
| `internal/service/sessions.go` | `services/auth/sessions.go`, `models/auth/session.go` | Session list/revoke/active-check, not a Gitea web session; stateless access JWTs remain valid until expiry |
| `internal/service/organization.go` | `models/organization/org.go`, `org_user.go` | Organization persistence and authorization. Team/Invite are newly added in R19 |
| `internal/api/response/*` | `routers/api/response/*` | Model-owned errors are mapped with `errors.Is`; safe JSON and logging remain the protocol owner |
| `internal/api/handler/resources.go` | `modules/structs/*`, `services/convert/*` | Public DTOs and conversion; ensure contract parity when adding endpoints |
| `internal/config/*` | `modules/setting/*` | INI configuration owner; database pool policy remains a separate review |
| `internal/errors/*` / `modules/errors/*` | `models/<area>/error.go`, `modules/util/error.go` | Domain-specific errors and shared classifications without a global PGX error map |

## Team/Invitation scope

The R19 migration adds `teams`, `team_members` and `team_invites` with
unique constraints and foreign keys. SQLC statements enforce organization
membership and owner checks, and `AcceptTeamInvite` consumes a hashed proof
and creates TeamMember in one transaction. Integration tests cover permission,
duplicate names, replay and concurrent acceptance.

No HTTP Team/Invitation endpoints or durable invitation mail delivery are
claimed in R19. `team_repo.go` is not added because BDSPro does not manage
Git repositories. Shared asset/project access requires a separately defined
real domain and permission contract.

## Outstanding work (do not mark as implemented)

- Move OTP and phone-signup persistent business rules from `models/db`
  into `models/auth` without changing commit-on-failed-code semantics.
- Move remaining Google identity operations from `models/db` into
  `models/auth` with the same atomicity and Google-subject uniqueness.
- Define HTTP endpoints, OpenAPI contracts and permission acceptance for Team
  before declaring the feature externally usable.
- Complete durable invitation delivery, lifecycle/revocation policy and
  operational observability before advertising production invitation support.
- CRM, Asset, Project, Deal and other future business domains need explicit
  migrations/queries/models: they are **not restored** by renaming `internal`.
- Do not create empty `modules/timeutil`, `models/perm`, or
  `team_repo.go` merely to copy Gitea's directory inventory.

Full acceptance requires PostgreSQL 18 integration, Go 1.27.3 race tests,
and reproducible SQLC output. A green CI does not prove unimplemented flows.
