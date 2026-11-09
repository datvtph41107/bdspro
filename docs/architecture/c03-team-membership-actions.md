# C03.2 — Team membership programming model

## Source and stack
- Parent PR #194, commit `96a632fbcf409ce8240a03fdde87ee162dc84266`; C03.1 owns typed form binding and Team listing.
- Gitea: `routers/api/v1/org/team.go` has `GetTeamMembers`, `AddTeamMember`, `RemoveTeamMember`. `services/org/team.go` coordinates writes, `models/organization/team_user.go` persists Team membership.
- Reference: https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/api/v1/org/team.go

## Owner and differences
- `routers/api/v1/api.go`: original GET/POST/DELETE routes remain, with typed `reqToken` and POST form binding.
- `routers/api/v1/org/team_member.go`: standalone actions `ListTeamMembers`, `AddTeamMember`, `RemoveTeamMember`; only path, form, HTTP errors and DTO.
- `models/organization.ListTeamUsersForDoer`: read the verified actor and Store from the request context, then execute existing account-scoped SQLC list. Avoid a redundant team-assignment query, because the list and zero-row visibility check already guard access.
- `services/organization.AddTeamMember`/`RemoveTeamMember`: `db.WithTxValue`/`db.WithTx`, lock live acting session first, then existing SQLC owner/recipient predicates. Audit deletion is transaction-scoped.
- One shared private pair of TX functions is used by both the new service functions and the historical receiver methods (retained only for still-running integration tests). Do not duplicate lock/permission logic.

## Acceptance
- Real Gin/PostgreSQL: owner add 201, member add 403 and delete 404 (SQLC DELETE intentionally maps zero rows to member-not-found), duplicate add 409, member list 200, outsider list 404, owner delete 204, physical membership row absent, duplicate delete 404.
- Negative unit tests: invalid path/pagination fail before DB; absent verified actor or Store explicitly fails.
- Existing pinned-connection concurrency tests now execute the new standalone service path for session revocation vs member writes, preserving original order.
- No change to routes, SQLC migrations, OAuth bearer audiences, public DTO or error code taxonomy.
- **Performance not yet measured**: empty list may perform a visibility check; no speed improvement claimed.

## Non-goals
- Team invitation uses proof, replay and outbox; distinct verification required.
- Do not reproduce Gitea User/Organization identity or repository auto-watch behavior in BDSPro's Account–Person–Organization model.
- Do not change nested transaction policy without rollback-only tests.
