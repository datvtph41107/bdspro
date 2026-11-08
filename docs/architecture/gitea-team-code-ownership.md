# Gitea-inspired team API: function ownership, resource visibility and query budget

Reviewed sources:
- BDSPro `dev@eb8cf9987db9b31fcdec44b646be3b2a7f7d0b67`
- Gitea `main@edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76`
- [Gitea routers/api/v1/org/team.go](https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/routers/api/v1/org/team.go)
- [Gitea services/context/api.go](https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/context/api.go)
- [Gitea models/organization/team_user.go](https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/models/organization/team_user.go)
- [Gitea services/org/team.go](https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/org/team.go)

## Verified differences and choices

| Question | Gitea | BDSPro choice | Why |
| --- | --- | --- | --- |
| HTTP functions | Package functions receiving `*context.APIContext` | Methods on a once-constructed `TeamHandler`, receiving `*appctx.APIContext` through `appctx.Wrap` | Store and invite signing key stay explicit; no process-global DB lookup or duplicate router/middleware executor |
| Context resource assignment | Context may hold `Org`/`Repo`/user models prepared before handler | For list routes, SQLC filters by the actor directly; resolve resource on zero-row result | Avoid a second SQL request on the common non-empty page, while preserving a 404 for invisible/missing scoped resources |
| Service ownership | `services/org` orchestrates repo permissions, audit, multiple models | Team membership rules stay in `models/organization` and owner-scoped SQLC predicates | A pass-through `services/org` would add no transactional/operational value |
| DB operations | XORM with DB session stored in context | SQLC and explicit `ExecTx(ctx, func(*db.Queries) error)` | No implicit transaction or OR-mapper replacement; owner predicates and audit remain in one transaction where required |
| Pagination | Shared HTTP list option extraction | One `teamListOptions` helper for teams and members | One bounded, validated request policy; SQLC offsets are checked again by `GetSkipTake32` |
| Errors | `ctx.APIErrorAuto`, model-owned error types | `ctx.APIErrorAuto`, stable public error codes | No PostgreSQL/driver strings reach the client |

## Request execution and ownership

```text
Gin routing (declared once)
  -> existing RequestIngress / Authenticate / RequireSession / VerifiedContact
  -> appctx.Wrap(method) [no extra middleware engine]
  -> TeamHandler (parse, validate, call model, respond)
  -> models/organization (owner/member authorization and business rules)
  -> models/db SQLC (actor-scoped SELECT/INSERT/DELETE; ExecTx when auditing)
  -> services/convert (response projection)
```

The same `TeamHandler` instance is registered for both organization-scoped
and team-scoped routes. It is created **once at router construction**, not per
request. This reduces duplicated initialization but does not claim
meaningful per-request throughput improvement.

## Zero-row resource ambiguity and performance

`ListTeamsForAccount` and `ListTeamUsersForAccount` include the actor's
active organization-membership predicate in SQL. Previously both an invisible
resource and a legitimate empty page yielded `200 []`. That did not
expose private member data, but it contradicted the existing resource
visibility contract (`GetTeam`/`GetForAccount` return 404).

The model now performs a scoped `GetTeam` or `GetForAccount` **only when
the list SELECT returns zero rows**, preserving the fast path:

| Outcome | SQLC operations | HTTP result |
| --- | ---: | --- |
| Authorized, nonempty page | 1 | 200 with list |
| Authorized, empty page (including out-of-range page) | 2 | 200 with `[]` |
| Missing/invisible resource | 2 | 404, same public error as single resource read |
| List SELECT fails | 1 attempted; no fallback | Existing sanitized 5xx |

No new table, index, join on the nonempty path, or cached allow decision is
introduced. This is a *query-count* budget, **not** a measured p95/p99
throughput claim. The empty-path query cost must be sampled under real
traffic before optimizing it further.

Two independent PostgreSQL reads do not promise a single serializable
authorization snapshot under concurrent membership changes. These are
read-only list endpoints; domain mutations retain their own SQL owner
predicates and transaction locks.

## Acceptance

- Unit tests assert SQLC list/visibility call counts and error mapping;
  malformed page/limit inputs stop at the HTTP boundary.
- Disposable PostgreSQL tests prove authorized empty page/list, invisible
  team and missing team return the intended results.
- Full Gin + PostgreSQL + Redis E2E verifies 200 empty for an authorized
  user and 404 for an outsider at both team and organization team-list URLs.
- Same-head CI must run gofmt, SQLC generation check, migrations, Go tests
  with race detector, route inventory, Asynq crash/recovery, Docker and
  HTTP smoke before merge.

## Non-goals

Do **not** replace Gin with Chi, duplicate Gitea's modules/web/router, add
`Org`/`Repo` fields to every APIContext, introduce a trivial pass-through
service, or change the owner-scoped SQLC mutation queries simply to match
Gitea's XORM syntax. Gitea is an operational reference, not a file template.
