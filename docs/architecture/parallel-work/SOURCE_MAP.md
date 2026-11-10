# Source Mapping — Gitea Reference → BDSPro Ownership

**A source map is not a proof of parity.** Each worker must follow imports, function callers and HTTP/transaction execution paths before accepting a change. Gitea main changes; pin SHA on each PR.

| Architecture axis | Gitea reference | BDSPro source / owner | Intentional non-parity | Acceptance question |
|---|---|---|---|---|
| HTTP route declaration | `modules/web/router.go` — `NewRouter`, `Group`, `Methods`, `BeforeRouting`, `AfterRouting` | `modules/web/router.go` — A | Gin not Chi | one matcher and one handler compilation path? |
| HTTP invocation | `modules/web/handler.go` — `toHandlerProvider`, `executeMiddlewaresHandler` | `modules/web/handler.go` — A | Gin boundary adapter only when behavior needs it | does committed response stop continuation with wrapped writer? |
| Pre-middleware contracts | `modules/web/types/premiddleware.go` | `modules/web/types/*` — A | no functional need to copy every Gitea provider | precedence across all three scopes? |
| Base and API context | `services/context/{base,api}.go` | `services/context/{base,api,gin}.go` — A | existing Gin request source temporary until migration | one context; cleanup; request cancellation and actor? |
| Endpoint registration | `routers/api/v1/api.go` | `routers/api/v1/api.go` — D after gate, A lease if needed | domain and public API differ | no unintended method/path/status/CSRF/redirect changes? |
| Persistence and Tx | `models/db/context.go` and domain model call sites | `models/db/context.go` + domain models — B | SQLC/pgx not Gitea ORM | tx-bound executor, nested rollback, verified session? |
| Business workflows | `services/{domain}` | `services/user/**` and domain services — B or D by approved owner | business domain differs | correct model vs service boundary, no HTTP leakage? |
| Lifecycle/init | `cmd/**`, `routers/init.go`, `modules/{setting,log,queue,cache,graceful}` | `cmd/**` and named `modules/**` — C; `routers/init.go` — 0 | Gin, app.ini, existing worker and infra | one init path, failure policies, resource shutdown? |
| Integrated trust and validation | Gitea tests and source links | CI, tests, scripts, PR acceptance — 0 | no fixed Gitea CI/template copy | every final HEAD and merged dev verified? |

## Reference SHA snapshot at initial P01 review
- Gitea `modules/web/router.go` blob: `2ddfbdaec67480fb6266e41f374c6ad73f033c3c`.
- Gitea `modules/web/handler.go` blob: `62fb0709b452be9b2ff46c4b8031faff28de3848`.
- Gitea `services/context/api.go` blob: `c14c3b3e467589904e4e3207cde1f4f65bca2140`.
- Recheck current `main` and use commit permalink; these SHAs are historical snapshot evidence only.

The link between the BDSPro and Gitea codebases is **function purpose and relationship**, not same business entity, persistence library, route framework, or unreviewed helper copying.
