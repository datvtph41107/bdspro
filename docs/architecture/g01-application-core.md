# G01 — BDSPro application core aligned with Gitea

Status: implemented on `dev`, 17 November 2026. This document records
source ownership and tested runtime behavior. Gitea reference revision:
`bf5c39b310f99b4254266bebee535d3730daba0a`.

## Dependency ownership

```
routers/init.go                 # composition root: DB/Redis, JWT, HTTP
  └─ routers/api/v1/api.go     # Chi route tree, binding, guards
       ├─ routers/common/auth.go       → services/auth/access.go
       │                                  ├─ modules/auth (JWT)
       │                                  ├─ models/auth/actor.go
       │                                  └─ models/db (session SQLC)
       ├─ routers/api/v1/org_assignment.go → services/org/organization.go
       ├─ routers/api/v1/ipactive.go   → modules/ipactive (Redis limiter)
       └─ routers/api/v1/org/team.go   → services/org/team.go
                                          ├─ models/organization (rules)
                                          └─ models/db (SQLC/transaction)
services/context/api.go        # APIContext stores verified Doer/Org
services/context/base.go       # request/response, Form*, JSON, Status
services/context/api_pagination.go # Link and X-Total-Count
modules/setting/api.go         # API pagination, body size, SwaggerURL
modules/util/optional.go       # small generic OptionalArg helper
modules/optional/option.go     # Some/None, JSON field presence
```

The HTTP context must not directly load sessions, verify JWTs, execute
Organization queries or apply business policies. It stores trusted state from
router actions and exposes HTTP response capabilities.

## API call sequence and trust boundaries

```
NormalHandler() // borrows Redis, sets IPActive request capability
  → web.Router (Chi, trusted ClientIP)
    → context.APIContexter() // request Data + deadline + writer
    → common.APIAuth()       // JWT signature + active DB session
    → orgAssignment()        // membership + Organization resource
    → reqOrgOwner()          // reject non-owners before state changes
    → reqIPAdmission()       // per-IP Redis counter, fail closed
    → bind(structs.CreateTeamOption{})
    → org.CreateTeam(ctx)
      → services/org.CreateTeam(context.Context)
        → models/organization.NormalizeTeamName
        → models/db.WithTx2
          → SQLC CreateTeamForOwner // role checked again under lock
    → ctx.JSON(201, team)
```

For list requests the handler invokes
`utils.GetListOptions(ctx)` → `db.ListOptions` →
`services/org.ListTeams` → `CountTeamsForAccount` and
`ListTeamsForAccount` with the same SQL membership predicate.
The handler emits `Link`, `X-Total-Count` and `Access-Control-Expose-Headers`.

PATCH form `RenameTeamOption.Name` is `optional.Option[string]`.
Omission becomes None, present empty string becomes Some(""), and JSON null
is rejected. Domain validation still determines whether a name is acceptable.

## API error and documentation contract

`ctx.APIErrorNotFound(msg...)` delegates to `util.OptionalArg`;
`ctx.APIErrorAuto(err)` accepts only explicitly public domain errors, and
unexpected errors remain masked. The envelope is `{error, code, url?}`.
`url` is configured by `setting.API.SwaggerURL` only in development;
Swagger UI is not enabled in production by this configuration.
JSON body reads are capped by `api.MAX_REQUEST_BODY_BYTES`.

IPActive uses source addresses resolved from trusted HTTP ingress, not raw
X-Forwarded-For; Team create/rename policies have distinct Redis keys.
Redis failures produce HTTP 503 instead of disabling rate limits.
Team endpoints return 429 with Retry-After when denied.

## Decision boundaries

NÊN: one typed HTTP APIContext, one SQLC transaction owner, explicit resource
assignment, explicit owner checks plus SQL recheck, no double JSON encoding.

KHÔNG NÊN: reimplement JWT or SQLC inside `services/context`, copy Gitea
session.KeyUID, render HTML through APIContext, trust client-supplied identity
or forwarded IP.

CHƯA NÊN: full Gitea repo/unit permission system and server-side HTML
renderer. BDSPro currently has Organization owner/member roles, not Gitea
repository permissions. Full end-user login/token issuance is still absent.

## Acceptance

GitHub Actions verifies gofmt, Go build/vet, SQLC generation, migrations,
Docker and a disposable PostgreSQL/Redis HTTP scenario:
- signed session owner and active Organization membership
- create/read/rename, member forbidden, duplicate name conflict
- paging query validation, SQL count, Link/total metadata
- absent/empty/null optional PATCH, invalid and oversized JSON
- rate-limit 429 and session revocation 401

The repository does not currently contain Go `*_test.go` files, as they
were removed at explicit request. CI smoke is not a substitute for
concurrency, authorization and security regression tests before production.
