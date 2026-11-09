# P0 — Gitea programming-model baseline and source ownership

**Status:** source-grounded inventory and a forward-only migration guard. This is **not** a full architecture conformance verdict.

## Exact source revisions

- BDSPro dev at b0d3b53800a59961b1ff2248a23d739e3b228053, running Go 1.27.3, Gin, SQLC/pgx and PostgreSQL 18.
- Gitea main at 0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a (a reference, not an alternative runtime).

Read the corresponding upstream code, not merely their folder names:
- [Gitea routers/init.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/init.go): mustInit, InitWebInstalled, NormalRoutes.
- [Gitea modules/web/router.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/modules/web/router.go): Group, BeforeRouting, AfterRouting, Methods, Mount, typed middleware, forms.
- [Gitea services/context/api.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/context/api.go) and [base.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/context/base.go): request-scope context and response lifecycle.
- [Gitea routers/api/v1/api.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/api/v1/api.go): resource grouping, admission and permissions.
- [Gitea services/org/team.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/org/team.go) and [models/organization/team_user.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/models/organization/team_user.go): transaction orchestration versus persistence.
- [Gitea modules/setting/setting.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/modules/setting/setting.go): configuration initialization phases.

## Observed BDSPro migration debt (exact baseline)

| Source | Baseline | Correct target |
| --- | --- | --- |
| routers/api/v1/api.go | 60 direct Gin method registrations, 5 typed registrations, 19 appctx.Wrap calls | One reviewed routing convention; no new raw Gin routes while converting existing registrations |
| routers/api/v1/org | 20 methods on *Handler/*TeamHandler (10 Organization, 10 Team) | Plain domain HTTP actions, with legacy receiver methods removed after their consumers and tests migrate |
| routers/init.go and routers/app.go | InitWebInstalled constructs mail transport; NewServer creates JWT, IPActive, Auth, OAuth/OIDC, handlers and Gin routes | Explicit dependency creator, consumer, error owner and cleanup path |
| modules/web/router.go | Small generic typed Router over Gin; not a Chi routing engine | One Gin-backed composition facade with proven guard order; no fake Chi BeforeRouting semantics |
| services/context | Base -> RequestSource -> Gin; APIContext.Doer = JWT claims; VerifiedActor = live-session actor | One request scope, committed response contract, no separate actor authority |
| services/organization | Context-scoped operations and Store/session-parameterized legacy operations coexist | Migrate callers into transaction-aware services; avoid two business algorithms |
| models/organization | Domain structs and generated db rows coexist | Define representation boundaries per operation; protect SQLC owner/session predicates |
| models/db/context.go | WithTx intentionally rejects nested transactions | Preserve rollback/cancellation semantics until an explicit new contract is proven |
| modules/setting | Typed full settings plus command-scoped worker/maintenance settings | Validate per command, not via the web server configuration for every CLI |

The counts are **transitional debt**, not a count of distinct API endpoints. tools/r02audit, OpenAPI, HTTP acceptance and SQLC tests remain the authority for actual external behavior.

The architecture regression test at tests/architecture/programming_model_test.go prohibits increasing 60/19 routing legacy counts or adding another Org HTTP receiver method; it requires all new v1 route registrations to stay in api.go and contains an intentionally failing synthetic change to verify the checker. It is deliberately limited: it does not infer full call graphs, dynamic route effects or transaction correctness. Review the check itself when changing the router facade.

## Canonical responsibility and naming rules

| Owner | Responsibilities | Must NOT own |
| --- | --- | --- |
| cmd | runWeb, runWorker, resource lifetime, startup/shutdown errors | HTTP business rules, domain authorization |
| modules/setting | INI provider, typed parsing, validate command-specific settings | DB or Redis connections, global mutable Store |
| modules/web | HTTP grouping, typed action adaptation, form binding with real consumers | Domain SQL or a second middleware executor |
| routers/init.go, routers/app.go | HTTP subsystem composition, Gin ingress policy, one route installation | Hidden work in business models or a general dependency locator |
| routers/api/v1/api.go | Paths, method/guard order, assignment and route registration | SQLC transactions or secret signing algorithm |
| routers/api/v1/{org,user,oauth} | HTTP actions, protocol validation and one response | Mutating SQL outside services/models |
| services/context | Base, APIContext, live actor, resource assignment and HTTP error contract | Persistent global credentials, SQLC domain writes |
| services/{organization,auth,oauth} | Business operations, transaction and outbox lifecycle, protocol rules | Gin parsing and public HTTP status codes |
| models/{organization,account,auth} | Domain entities, authorized SQLC operations, persistence invariants | Gin, HTTP error envelopes |
| models/db | Store, SQLC-generated code and transaction mechanics | Organization or OAuth policy |
| modules/{conn,queue,cache,log,auth} | Technical capabilities with consumers | Empty Gitea-lookalike packages |

A read with verified authority may call a model directly. A write needing session/owner concurrency control goes through a service transaction. The BDSPro acting identity (Account with live PostgreSQL Session) must **not** be replaced by Gitea's User pointer or an OAuth access token merely to match a type name.

Gitea mustInit logs fatal startup failures; it does not provide automatic rollback. BDSPro keeps errors, defers and resource ownership in cmd. Gin middleware operates on its own routing lifecycle, not on Chi's BeforeRouting/AfterRouting phases.

## P1 through P5: required evidence before declaring conformance

1. **P1: modules/web + routing** — compare the current Gin grouping with a typed facade in a real Gin integration test: auth admission, group guard ordering, 404/405, abort, committed response, route inventories and benchmark. Select one supported API convention; remove parallel registration style only after parity.
2. **P2: installed runtime** — draw the actual constructor, consumer, failure, cleanup edges for cmd/runWeb, routers.InitWebInstalled, NewServer, Redis, DB, queue. Keep R01.9 open; PR #192 is a separate HTTP shutdown experiment awaiting review.
3. **P3: Base, APIContext, security** — prove one context and response per request; separate session authentication, OAuth authorization-server scopes, Organization visibility and in-transaction owner predicates. Draft PR #193 is only a response lifecycle slice.
4. **P4: models and services** — resolve duplicate Organization/Team operations, representation and transaction scope. Draft PRs #194 then #195 are a stacked consumer migration, not proof of total refactor. No SQLC nested-transaction rewrite without rollback-only tests.
5. **P5: acceptance** — run PostgreSQL race/revocation tests, inspect SQL query counts and allocations before/after, keep Go 1.27.3, SQLC parity, OpenAPI, worker failure recovery and Docker runtime smoke. Only reduce the debt ceilings after verified migrations.

### Engineer and agent contract

Each PR must identify the pinned before/after source, owner in package/file/function, callers/callees, public/permission/transaction/failure contracts, a negative test that would detect regression, exact green CI commit, and remaining uncertainty. No PR should claim complete Gitea-style conformance merely because tests pass or directory names match.
