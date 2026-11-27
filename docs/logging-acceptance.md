# Logging stabilization and acceptance

Scope: IAM feature branch only. Do not merge into main on source review or an isolated lab alone.

## Decisions

| Mechanism | Decision | Evidence / next gate |
| --- | --- | --- |
| Compact console and short display ID | Keep | Direct developer readability requirement; full UUID in response and JSON |
| Destination levels and component filter | Keep | Separate presentation from data retention; filter tests and race test |
| Caller hook | Conditional | Validate source locations in actual logger integration; measure runtime-frame cost before further expansion |
| Handler activity | Conditional | Validate guard, unmatched route and typed handler metadata |
| Slow timer | Trial | Per-request allocation and synchronous output delay are costs; no evidence yet for a shared registry; retain until real usage and measurement |
| Safe error chain | Keep | Preserve errors.Is/As and operation names; never serialize raw error strings in these diagnostics |
| Stack frames | Conditional | Bounded reporting-point stack, not the origin of a returned error |
| HTTP panic boundary | Fix | Log safe metadata then panic(http.ErrAbortHandler); intentional abort is WARN handler_abort |
| Delve and offline profiles | Keep | Developer tooling without runtime debug endpoints |
| Named writer registry, async queue, runtime controls | Defer | No demonstrated requirement |

Panic conversion is only safe at the final request boundary: net/http suppresses its raw panic diagnostic for the exact sentinel and interrupts the connection/stream. It does not promise a 500 response. It cannot protect panics already logged by other middleware, startup/background panics, or arbitrary log calls outside this path. The regression test uses an actual httptest server for HTTP/1.1 and HTTP/2 before and after partial writes.

## Reproducible CI

The IAM logging acceptance workflow runs on pushes to feature/iam-v1-implementation. It reads go.mod for the toolchain, verifies modules, builds, runs vet and full Go tests, runs focused race tests, starts the application with disposable PostgreSQL and Redis, correlates four HTTP responses with the JSON file, checks normal process shutdown and runs the formatter benchmark. No repository secrets, external provider or production database are used. GitHub-hosted runner availability and dependency resolution are prerequisites; workflow creation alone does not prove success.

A successful smoke test does not prove authenticated login/session creation, 429/503 behavior, all privacy fields, or performance at production load. Some existing Go integration tests may skip without their own opt-in environment. Review job logs and skip reports before claiming coverage. Benchmark results are observations, not a release latency threshold.

No automated merge or release. A passing workflow is necessary, not sufficient: review privacy, actual IAM use, costs and local Delve separately. Record exact commit and CI run URL for the acceptance decision.
