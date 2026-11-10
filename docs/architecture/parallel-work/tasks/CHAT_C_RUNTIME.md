# Chat C — Runtime Modules, Initialization, Operations (P06)

**Role:** owner of independent process and capability packages. **Branch:** `work/gitea-c-runtime-20261110` starting from current `dev`.

## Read sources
- Shared [README](../README.md), [ownership](../OWNERSHIP_AND_CONTRACTS.md), [trust](../TRUST_AND_VERIFICATION.md), [integration](../INTEGRATION_PROTOCOL.md), [skill](../SKILL.md).
- Gitea `cmd`, `routers/init.go`, `modules/{setting,log,cache,queue,graceful}` actual files and consumers; Gitea init sequencing is the reference.
- BDSPro `cmd/main.go`, `routers/init.go`, `modules/setting/setting.go`, `modules/cache/cache.go`, `modules/queue/queue.go` and actual log/graceful equivalents. **Do not assume a Gitea filename exists in BDSPro**. Search source first and write only usable code.

## Ownership
- **Allowed:** `cmd/**`, `modules/setting/**`, `modules/log/**`, `modules/cache/**`, `modules/queue/**`, `modules/graceful/**`, each only where consumed. A module not currently present must be justified by an actual caller.
- **Read only:** `routers/init.go`, Auth services, routers/API, `models/db`, `modules/web`, `services/context`.
- **Cross-owner:** new constructors/dependency injection via `routers/init.go` or `go.mod` require written interface contract and coordinator-owned integration patch.

## Milestones
- C1: inventory process entrypoints and init ordering. Compare against Gitea code and actual module imports. Categorize KEEP/MODIFY/REMOVE/DEFER with evidence.
- C2: verify configuration `app.ini` source, logging initialization and cleanup, lifecycle error policy; remove duplicate connection/setup only if existing code is truly redundant.
- C3: harden queue/cache/graceful behavior with real consumers. Redis startup/failure, worker lease and shutdown behavior must be explicit; no untested best-effort security degradation.
- C4: propose a small stable constructor/initialization interface to Chat 0 for wiring in `routers/init.go`, without touching that shared file directly.
- C5: full CI and runtime evidence, retained CLI `serve/migrate/worker/help` and Makefile targets, graceful shutdown, recovery; handoff.

## Explicit exclusions
No replacement Gin/SQLC, no new framework ports solely to imitate Gitea; no editing `routers/init.go`, production route declarations, DB schema or Auth/OAuth flow. Do not delete Makefile migration/sqlc/swagger targets to simplify architecture.

## START NEW CHAT — copy the following message

> Bạn là **Chat C — P06 Runtime/Modules** của BDSPro. Repository https://github.com/datvtph41107/bdspro ; Gitea https://github.com/go-gitea/gitea ; branch `work/gitea-c-runtime-20261110` dựa trên `dev`. Đọc `docs/architecture/parallel-work/README.md`, `OWNERSHIP_AND_CONTRACTS.md`, `TRUST_AND_VERIFICATION.md`, `INTEGRATION_PROTOCOL.md`, `SKILL.md` và `tasks/CHAT_C_RUNTIME.md` trên nhánh docs `docs/gitea-parallel-harness-20261110`. Bạn chỉ được viết `cmd/**` và các package `modules/setting/log/cache/queue/graceful/**` có consumer thực. `routers/init.go` do Chat 0 quản lý, cần RFC/contract nếu có thay đổi; không tự edit. Phân tích code Gitea về tên file, hàm, import và lifecycle, xác minh BDSPro live source; triển khai có tests, thực thi CI, đánh giá failure modes và cung cấp PR/handoff. Giữ Go 1.27.3, Gin, SQLC, app.ini, Redis/Asynq và command behavior. Không merge dev.
