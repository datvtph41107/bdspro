# Organization/Team API: an end-to-end Gitea-style vertical slice

**Status:** implemented on `dev` at the current source version. This is the
first real business API after the H01 Chi-only HTTP infrastructure migration.

## Routes

| HTTP | Path | Access | Meaning |
|---|---|---|---|
| GET | `/api/v1/health` | public | infrastructure HTTP probe |
| GET | `/api/v1/orgs/{org}/teams?offset=0&limit=20` | organization member | paginated Team list |
| GET | `/api/v1/orgs/{org}/teams/{team}` | organization member | one Team |
| POST | `/api/v1/orgs/{org}/teams` | organization owner | create Team |
| PATCH | `/api/v1/orgs/{org}/teams/{team}` | organization owner | rename Team |

Create body: `{"name":"Engineering","description":"Backend"}`.
Rename body: `{"name":"Platform"}`.

## Concrete call graph

```
routers.NormalHandler
  → web.Router / Chi
    → apiv1.Routes() → TeamRoutes()
      → context.APIContexter()          // request scope, response writer
      → context.AuthenticateAccessToken // verified Bearer JWT AND DB session
      → context.AssignOrganization      // SQLC membership lookup
      → bind(CreateTeamOption{})        // validation, web.SetForm
      → apiv1.CreateTeam               // web.GetForm[*CreateTeamOption]
        → services/org.CreateTeam       // context.Context, no HTTP imports
          → models/organization.NormalizeTeamName
          → models/db.WithTx2           // single commit/rollback authority
            → models/db.GetQueries(txCtx)
              → SQLC CreateTeamForOwner // locks active owner membership
        → context.APIContext.JSON(201)
```

The code is intentionally split into:
- `routers/api/v1/team.go`: HTTP route declaration and status mapping only
- `services/context/organization.go`: request-scoped verified Actor and Org
- `services/org/team.go`: business orchestration using context-first SQLC
- `models/organization/team.go`: domain value and safe error contracts
- `db/query/team.sql`: PostgreSQL permission checks/locking (existing query)
- `models/db/context.go`: actual transaction commit/rollback (existing core)

## Security and authorization

The client must supply an **existing session-bound access JWT**, issued by
`auth.Maker.CreateSessionAccessToken`. A signed legacy JWT without session_id
is rejected by these endpoints. The session UUID must exist in PostgreSQL,
be unblocked and unexpired, and belong to the signed account ID.

A caller cannot select an account using `X-User-ID`, request parameters, or
JSON. Organization membership is looked up through SQLC with the verified
actor. SQL writes independently check **active owner** membership under
PostgreSQL row locks; possession of a member token does not grant writes.
Other organizations are not exposed by direct Team ID access.

The current repository does **not** yet expose an end-user login/token-issuance
endpoint. These protected routes therefore require an access JWT issued by a
separate existing trusted authentication flow or a developer fixture tied to
a real database session. Do not add a development-only bypass to production.

## Example commands

Start the application with your configured PostgreSQL, Redis and migrations.
Set `TOKEN` to a real, active session-bound access JWT. Replace IDs:

```bash
export BASE=http://127.0.0.1:8080/api/v1
export ORG_ID=1
export TEAM_ID=1
export TOKEN='paste-real-session-access-token'
curl -i -H "Authorization: Bearer $TOKEN" "$BASE/orgs/$ORG_ID/teams"
curl -i -H "Authorization: Bearer $TOKEN" "$BASE/orgs/$ORG_ID/teams/$TEAM_ID"
curl -i -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Engineering","description":"Backend"}' "$BASE/orgs/$ORG_ID/teams"
curl -i -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Platform"}' "$BASE/orgs/$ORG_ID/teams/$TEAM_ID"
```

Without authorization: 401. Missing org membership: 404. Invalid JSON/form:
400. Member attempting a write: 403. Duplicate team name: 409.

## Runtime acceptance and scope

CI builds all packages, runs go vet, checks SQLC/migrations and Docker, and
smoke-checks health plus unauthenticated Team API denial. The repository's
Go regression tests were previously removed at explicit request; a green
smoke test cannot establish that all authenticated or concurrent behaviors
have been tested. Reintroduce security, authorization and transaction
regression tests before deploying business APIs to production.
