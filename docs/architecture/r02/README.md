# R02 — Gitea-aligned backend implementation

This is the **implementation contract**, not a claim that the refactor has shipped.

- BDSPro source baseline: `datvtph41107/bdspro@e1c3085321a26ee244809be80977155b2a10df1d` (R01, PR #91).
- Original `dev` at planning time: `e88619a3b42553b3cbea58259378d57fe25c6cdc`.
- R00, R01 and R02 foundation merged to `dev` on 2026-11-03; current audited merge commit: `36af7f7eadb321ff5ae02bbbd2398511f96c6ce2`.
- Gitea reference: `go-gitea/gitea@12b20c7c9962e0be75912b290a0c5778799b0ad2`.
- Existing public URLs, JSON shapes, statuses, security rules and database semantics must be characterized **before** migration.
- GitHub PRs #90, #91 and #92 have been merged after CI. **This does not mean that the full Chi cutover has shipped.**

Read:

1. [Code correspondence and allowed deviations](gitea-reference.md)
2. [PR implementation plan](implementation-plan.md)
3. [Acceptance and rollback gates](acceptance.md)
4. [Pinned R01 route inventory](routes-r01.json)

The route inventory contains 34 method/path pairs from `routers/api/routes.go` and `routers/api/swagger.go`. It includes conditional routes, so it is a **superset** of routes registered for a given configuration. Route group inheritance and runtime responses are not established by this file. The audit tool verifies source-level method/path drift during R01 stabilization:

```sh
go run ./tools/r02audit -root . -contract docs/architecture/r02/routes-r01.json
```

The tool is **temporary and Gin/R01-specific**; replace its source extractor with a runtime route inventory or new Chi extractor at R02-08, preserving `routes-r01.json` as the immutable historical contract. Never silently delete this contract to get a green test.
