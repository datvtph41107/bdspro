# Team and invitation model ownership

`models/organization/team.go` owns Team identity and owner-guarded writes.
`team_user.go` owns membership and the organization-member invariant.
`team_list.go` owns SQLC-backed, bounded list queries and query options.
`team_invite.go` owns hashed one-time invite proofs and atomic acceptance.

Schema: `db/migration/000012_teams.up.sql`.
Queries: `db/query/team.sql`.
Generated code: `models/db/sqlc/team.sql.gen.go` (do not edit).
HTTP response/error codes are a separate API-contract change.

Team membership never grants access if the user's organization membership
has ended. No Team ↔ Git repository relation exists in BDSPro, so Gitea's
`team_repo.go` is intentionally not copied.

Raw invitation proofs are returned only from the model operation to the
invoking service, which must arrange delivery and never log or persist the
plaintext. This PR does not expose an invitation endpoint or claim durable
delivery semantics.
