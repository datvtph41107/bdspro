# G1.2c.4 — Transaction-ordered team membership writes

Baseline: BDSPro dev@e58a652fbd53dc7bf736492eb43d8b02e2db9c66.
Gitea reference:
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/org/team.go
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/routers/api/v1/org/team.go

## Source-proven gap

HTTP RequireSession checks the first-party JWT session at request admission.
A concurrent revoke may commit before a later team INSERT or DELETE.
The existing team SQLC `FOR SHARE OF m` predicates correctly serialize
**resource ownership**, but do not serialize **acting session authority**.
These are independent requirements.

## Ownership

- `routers/api/v1/org.TeamHandler`: parse and validate request, take
  account/session IDs from signed `APIContext.Doer`, send the result.
- `services/organization.TeamService`: one transaction coordinating
  session authority and team membership mutation.
- `models/auth.RequireActingSession`: verify live session/account and
  hold the session row `FOR UPDATE` until the caller transaction commits.
  The profile service reuses this same model helper instead of duplicating it.
- `models/organization.AddTeamUser`: unchanged owner and recipient
  authorization predicates in SQLC; narrowed query dependency so it can
  execute on `db.Queries` bound to an outer transaction.
- `models/organization.RemoveTeamUserTx`: same audited owner-scoped
  DELETE, but nested transactions are avoided; the old standalone
  `RemoveTeamUser` delegates to this transaction operation.

## Ordering, correctness and costs

The service takes the **acting session row lock first**, then runs the
resource authorization SQLC statement. Revoke-one/revoke-all still take
their account `FOR NO KEY UPDATE` lock before changing session rows.
Membership writes do not update other session rows and do not require
a new account-level lock. Adding that lock would serialize unrelated
team mutations of the same account with no established benefit.

Two directions must be verified on disposable PostgreSQL using
`pg_blocking_pids` and pinned backends:
1. Revocation locks the acting session first. The team writer waits,
   then rejects; membership is unchanged.
2. Team write locks the acting session first, then waits for membership
   `FOR SHARE`. Revocation waits for the session lock; the team write
   commits first and revocation follows.

Existing membership revocation tests also remain required: neither
session authorization nor SQLC membership authorization can replace the
other. Database outages remain infrastructure errors, not 401s. An
audit write failure must roll back member removal.

Costs: INSERT gains one transaction and one row-lock SQL statement;
DELETE adds one row-lock SQL statement to its existing transaction.
Do not claim a p95/p99 improvement without load tests and lock metrics.

## Deliberate exclusions

G1.2c.4 covers only `POST /teams/:team_id/members` and
`DELETE /teams/:team_id/members/:person_id`. It does not yet close
commit-time session authority on create/rename team, organization
membership, invitations, accepted invitations, or OAuth client writes.
No new middleware, response type, Go router, SQLC generation or migration.

See PostgreSQL 18 explicit locks:
https://www.postgresql.org/docs/18/explicit-locking.html
