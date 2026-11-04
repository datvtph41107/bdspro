# Team and invitation model ownership

`models/organization/team.go` owns Team identity and owner-guarded writes.
`team_user.go` owns membership and the organization-member invariant.
`team_list.go` owns SQLC-backed, bounded list queries and query options.
`team_invite.go` owns hashed one-time invite proofs and atomic acceptance.

Schema: `db/migration/000012_teams.up.sql`.
Queries: `db/query/team.sql`.
Generated code: `models/db/sqlc/team.sql.gen.go` (do not edit).
Model errors map to explicit public response codes in `routers/api/response/errors.go`.

Team membership never grants access if the user's organization membership
has ended. No Team ↔ Git repository relation exists in BDSPro, so Gitea's
`team_repo.go` is intentionally not copied.

Raw invitation proofs are returned only from the model operation to the
invoking service, which must arrange delivery and never log or persist the
plaintext. This PR does not expose an invitation endpoint or claim durable
delivery semantics.

Domain errors in `models/organization/error.go` wrap a domain-neutral
`modules/util` category for `errors.Is` classification, following Gitea's
error layering. The HTTP response mapper exposes **only** explicitly listed
model errors; generic categories and unknown driver/provider errors remain
private. All Team errors have stable JSON `code`/status mappings and tests.
