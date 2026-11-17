# Team API — verified Gitea-style application slice

## Routes

| Method | Path | Policy |
|---|---|---|
| GET | `/api/v1/health` | public |
| GET | `/api/v1/orgs/{org}/teams?page=1&limit=20` | verified member |
| GET | `/api/v1/orgs/{org}/teams/{team}` | verified member |
| POST | `/api/v1/orgs/{org}/teams` | owner + IPActive team_create |
| PATCH | `/api/v1/orgs/{org}/teams/{team}` | owner + IPActive team_rename |

POST body: `{"name":"Engineering","description":"Backend"}`.
PATCH body: `{"name":"Platform"}`; optional name presence is explicitly
tracked; empty, missing and null are rejected by different validation checks.

## Real source caller/callee

1. `routers/init.go`: `NormalHandler` creates the IPActive Redis capability.
2. `routers/api/v1/api.go`: route declarations and action order.
3. `routers/common/auth.go` → `services/auth/access.go`: signed JWT plus
   current DB session become `ctx.Doer`.
4. `routers/api/v1/org_assignment.go` → `services/org/organization.go`:
   active membership becomes `ctx.Org.Organization`; owner rule evaluated.
5. `routers/api/v1/ipactive.go`: trusted client IP checked against Redis.
6. `modules/web/middleware/binding.go`: validated JSON put in request store,
   retrieved via `web.GetForm[*structs.CreateTeamOption](ctx)`.
7. `routers/api/v1/org/team.go` → `services/org/team.go` → SQLC under
   `db.WithTx2`; write SQL rechecks owner membership under lock.
8. `services/context/api.go`: stable masked error response and JSON output.

Pagination uses `utils.GetListOptions(ctx)`, `db.ListOptions`, and
`CountTeamsForAccount` to set `X-Total-Count` and standard Link headers.

Run the existing disposable scenario with:
`BDS_TEST_MODE=disposable BDS_TEST_DB_URL=... BDS_TEST_TOKEN_KEY=... bash scripts/demo-team-api.sh`.
It refuses databases other than `bdspro_test`. GitHub Actions also builds
and runs the application against a disposable PostgreSQL/Redis pair.

This API does not yet include login/token issuance. Do not add an
unauthenticated admin bypass as a shortcut.
