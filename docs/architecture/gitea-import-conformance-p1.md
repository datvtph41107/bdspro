# P1 — Gin router equivalence and Gitea package-import correspondence

**Status:** a falsifiable source/behavior comparison; not a full Gitea parity claim.

**Pinned baseline:** Gitea `0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a`, BDSPro P0 `766d653d7d7df9a1f65433c543173ab019649ae5` (PR #196).

## Gitea actual source import directions vs BDSPro

| Owner | Gitea code and imports | BDSPro code and imports | Contract, not copy |
| --- | --- | --- | --- |
| `routers/init.go` | `NormalRoutes() *web.Router`, imports `modules/web`, `routers/api/v1`, `services/auth`, `modules/setting` | `InitWebInstalled(cfg, store, redisClient)`, `NormalRoutes(server) *gin.Engine`; imports Gin, `modules/setting`, `routers/api/v1`, `services/context`, `services/security` | HTTP composition is owned by `routers`; don't substitute Gitea fatal `mustInit` for explicit errors/cleanup |
| `modules/web/router.go` | `Router` wraps Chi; imports Chi, `modules/reqctx`, `setting`, `validation`, `web/middleware` and `web/types` | `Router[T]` wraps Gin; `router.go` imports Gin, `form.go` imports `modules/reqctx` | Only technical dependencies; no model/service/router imports; no false Chi BeforeRouting semantics |
| `services/context/api.go`, `base.go` | Context imports user model, `modules/web`, `reqctx`, `setting`; knows API principal, request and response | `api.go` imports `models/auth` and `modules/auth`; `base.go` imports `modules/reqctx`; `gin.go` adapts Gin | Request context may know models and protocol, but must not orchestrate `services/organization`/auth business operations |
| `routers/api/v1/api.go` | Route composition imports domain `routers/api/v1/org,user,...`, `modules/web`, `services/context`, permission models | Current `Routes(root *gin.Engine, Dependencies)` imports `modules/web`, Gin, `routers/api/v1/{org,user,oauth}`, `services/context` | One route composer; authenticated guard ordering and errors must be behaviorally verified; no SQL mutations in route declaration |
| `routers/api/v1/org/team.go` | Standalone `CreateTeam`, `ListTeams`, `GetTeamMembers`; imports `models/organization`, `modules/web`, `services/context`, `services/convert`, `services/org` | `CreateTeam`, `GetTeam`; equivalent BDSPro imports `models/organization`, `modules/web`, `services/context`, `services/convert`, `services/organization` | Very similar responsibility: adapt HTTP to business/model operations. Still transitional because legacy receiver actions coexist |
| `services/org/team.go` | Business transaction/side effects; imports `models/db`, `models/organization`, `models/user`, repository service | `services/organization/team.go` imports `models/db`, `models/auth`, `models/organization` | Same orchestration direction; BDSPro session/Actor and SQLC predicates are authoritative, not Gitea User/repository auto-watch |
| `models/organization/team_user.go` | Imports `models/db`, `models/user`, XORM builder | Imports `models/db`, `models/audit`, `modules/util`, pgx | Persist via SQLC, not Gitea XORM; no Gin, router or business-service imports |
| `modules/setting/setting.go` | Imports `modules/log`, `modules/util`, and configuration support; initializes settings in phases | Typed settings split across `modules/setting/*.go`; CLI-specific load/validation | Settings own parsing and validation, not domain Store or HTTP handler construction |

Sources pinned:
- [Gitea routers/init.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/init.go)
- [Gitea modules/web/router.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/modules/web/router.go)
- [Gitea services/context/api.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/context/api.go)
- [Gitea routers/api/v1/api.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/api/v1/api.go)
- [Gitea org/team.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/api/v1/org/team.go)
- [Gitea services/org/team.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/org/team.go)
- [Gitea models/organization/team_user.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/models/organization/team_user.go)

## P1 acceptance contract

The test `TestTypedRouterNativeGinParity` constructs the **same groups independently** with native Gin and typed `modules/web.Router[T]`, then compares real `net/http` results and side-effect order:

1. Read and write success: nested group assignment -> membership -> owner -> action.
2. Failure precedence: assignment abort 404, membership/owner abort 403, with later handlers never invoked.
3. Unknown resource 404; wrong HTTP method 405 when `HandleMethodNotAllowed=true` (explicit Gin setting).
4. Both routers must emit exactly matching body and status in every case.

This proves a narrow, real behavioral equivalence on Gin for the tested composition. It **does not** prove equivalence with Chi's before-routing middleware, Gitea's `Mount`, production auth middleware, form decoding, 405 behavior when the production Gin flag differs, OAuth/session semantics, or performance.

`TestProgrammingModelImportBoundaries` guards direct import edges for `modules/web`, `modules/setting`, `services/context`, `services/organization`, and `models/organization`; a negative fixture must reject forbidden imports. It supplements—not replaces—`TestSourcePackageOwnership`.

## Next P1 production work (not included)

1. Decide `Routes()` return-vs-register API only after real native/typed route parity across current public route inventory, middleware order and request-scoped dependencies. Gitea's return signature is not itself proof of correctness.
2. Build the smallest Gin-native route composer that replaces mixed registrations **without a second runtime**, prove exact route/guard/API/error parity, then progressively migrate dependent actions with clean rollback.
3. Reduce P0 migration debt ceilings as handlers/receivers are actually retired; do not create empty adapter packages merely to make imports mirror Gitea.
4. Review open #193 (Base response), #194 (typed routing), and #195 (stacked membership) against this contract. HTTP and PostgreSQL acceptance is a separate gate.
5. Keep R01.9 initialization/cleanup work in #192 and subsequent runtime ownership review, rather than hiding it in P1.

**Engineering/agent rule:** require pinned source revision, package+file+function+caller evidence, expected import direction, negative test, actual CI SHA and independent review.
