# Evidence ledger — source anchors, contradictions, and claims

**State:** Seed ledger only. **Verified as snapshot:** 2026-11-20. Refresh HEAD, source lines and consumers before reusing. Source facts below do not approve any architecture.

## Source revisions (not permanent HEAD claims)

| Key | Repo / revision | Role |
| --- | --- | --- |
| BDS | [BDSPro dev](https://github.com/datvtph41107/bdspro/tree/abe04e46269632ef671a92bab00319b52696e8f5) · `abe04e46269632ef671a92bab00319b52696e8f5` | Implementation being audited |
| GIT | [Gitea](https://github.com/go-gitea/gitea/tree/fbf986d1c7ba1b0ad0078105b3adb0a73b769295) · `fbf986d1c7ba1b0ad0078105b3adb0a73b769295` | Go architecture comparator |
| DRX | [Laravel DRX](https://github.com/datvtph41107/laravel-drx/tree/767737d3a9530d0ec3ff94876752e40ccb302a15) · `767737d3a9530d0ec3ff94876752e40ccb302a15` | PHP architecture comparator |

## Fact ledger (only inspected slices)

| ID | Category | Narrow fact with permalink | Limits / next caller or consumer |
| --- | --- | --- | --- |
| S-B01 | SOURCE FACT | [BDS routers/init.go L55–86](https://github.com/datvtph41107/bdspro/blob/abe04e46269632ef671a92bab00319b52696e8f5/routers/init.go#L55-L86) `NormalRoutes()` mounts v1; `NormalHandler()` borrows Redis, constructs limiter, configures proxy trust and returns handler closure. | Whole call chain and production deployment not yet accepted as architecture. |
| S-B02 | SOURCE FACT | [BDS modules/web/router.go L97–110, L233–265](https://github.com/datvtph41107/bdspro/blob/abe04e46269632ef671a92bab00319b52696e8f5/modules/web/router.go#L233-L265) uses `chi.Mux` and `With(...).Method(...)` for matching. | Inspect `ServeHTTP`, mount behavior, Chi version and all route consumers before refactor. |
| S-B03 | SOURCE FACT | [BDS modules/web/router_chi.go L57–93](https://github.com/datvtph41107/bdspro/blob/abe04e46269632ef671a92bab00319b52696e8f5/modules/web/router_chi.go#L57-L93) resolves peer and conditionally parses forwarding headers when peer is trusted. | Proxy header sanitation/deployment unverified. |
| S-B04 | SOURCE FACT | [BDS services/context/http.go L132–159](https://github.com/datvtph41107/bdspro/blob/abe04e46269632ef671a92bab00319b52696e8f5/services/context/http.go#L132-L159) creates request-scoped APIContext, wraps body with MaxBytesReader and tracks response on matched-route `APIContexter`. | Body read and errors must be traced to binder/validator and real HTTP tests. |
| S-B05 | SOURCE FACT | [BDS routers/api/v1/api.go L16–27](https://github.com/datvtph41107/bdspro/blob/abe04e46269632ef671a92bab00319b52696e8f5/routers/api/v1/api.go#L16-L27) uses Group/Combo/Post actions; route guards precede bind and endpoint in declaration. | Preserve declared API contract; call graph and behavior to be tested. |
| S-G01 | SOURCE FACT | [Gitea API Routes L953–979](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/routers/api/v1/api.go#L953-L979) registers CORS in `BeforeRouting`, APIContext/auth in `AfterRouting`. | Does not imply BDS must copy CORS configuration or global routing structure. |
| S-G02 | SOURCE FACT | [Gitea modules/web/router.go L83–110, L149–188](https://github.com/go-gitea/gitea/blob/fbf986d1c7ba1b0ad0078105b3adb0a73b769295/modules/web/router.go#L149-L188) composes typed provider chain and registers into Chi. | Inspect adapter semantics and cost before any port. |
| S-L01 | SOURCE FACT | [DRX app/Http/Kernel.php L20–125](https://github.com/datvtph41107/laravel-drx/blob/767737d3a9530d0ec3ff94876752e40ccb302a15/app/Http/Kernel.php#L20-L125) separates global, API group and named route middleware. | This is Laravel runtime organization, not a Go implementation requirement. |
| S-L02 | SOURCE FACT | [DRX RouteServiceProvider.php L26–46](https://github.com/datvtph41107/laravel-drx/blob/767737d3a9530d0ec3ff94876752e40ccb302a15/app/Providers/RouteServiceProvider.php#L26-L46) maps API route file under `api/v1` and `api` prefixes. | Compare real route inventory and versioning cost before generalizing. |
| S-L03 | SOURCE FACT | [DRX IPActiveMiddleware.php L8–25](https://github.com/datvtph41107/laravel-drx/blob/767737d3a9530d0ec3ff94876752e40ccb302a15/modules/IPActive/src/Http/Middleware/IPActiveMiddleware.php#L8-L25) delegates request admission to `IPActiveService`. | Full write/atomicity/deployment comparison belongs to a separately scoped study. |

## Contradictions to resolve, not silently rewrite

- C-01 **SOURCE VS HISTORICAL DOCUMENT**: [docs/architecture/http.md](../http.md) and [docs/architecture/routers.md](../routers.md) claim Gin is the active matcher. BDS `abe04e4` above constructs Chi. Those docs require a scoped factual correction after an inventory; do not use them as runtime authority.
- C-02 **OLD PLAYBOOK**: [gitea-alignment/CONTINUE_PROMPT.md](../gitea-alignment/CONTINUE_PROMPT.md) includes Gin-era P01 instructions/branch history. Keep as historical record; not a mandate to recreate old branch/PR or reverse Chi work.
- C-03 **PARALLEL PLAN**: [parallel-work/README.md](../parallel-work/README.md) describes older multi-lane implementation. Current single-phase user instruction takes precedence; do not spawn those lanes without new authorization.
- C-04 **OLD F40 DIRECTION**: AGENTS and F40 files preserve an earlier request. No inference that F40 implementation is currently authorized or complete.

## OPEN, not architecture verdicts

- Q-01: Full HTTP ingress route, including `net/http.Server`, proxy topology, CORS, observation/recovery, 404/405, APIContext, resource/error handling: **TRACE INCOMPLETE**.
- Q-02: Best package owner for trusted IP and request metadata: **OPEN**; prior `modules/httpmeta` prototype is illustrative, not approved.
- Q-03: Per-request context vs constructor injection for process-scoped admission: **OPEN**; requires call-site, tests and migration cost.
- Q-04: Default CORS, body limits, upstream proxy, request logging, panic handling and operation/security policy: **OPEN** until specific threat model and integration tests.
- Q-05: Impact of any change on workers, CLI, DB transactions, tests, API surface and production deployment: **NOT INSPECTED / REQUIREMENTS PENDING**.

No active ADR in this ledger is accepted. Use [DESIGN_CASE_TEMPLATE.md](DESIGN_CASE_TEMPLATE.md) before promoting a proposal.
