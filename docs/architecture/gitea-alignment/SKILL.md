---
name: bdspro-gitea-alignment
description: Source-grounded implementation and acceptance workflow for BDSPro's Gitea-style Go architecture while preserving Gin/SQLC.
---

# BDSPro × Gitea Source Alignment Skill

This is a repository-local **skill-style playbook**. It is not installed automatically in ChatGPT or any external runtime. A new coding assistant should read it explicitly alongside [CONTINUE_PROMPT.md](./CONTINUE_PROMPT.md).

## Triggers

Use for any task involving BDSPro `modules/web`, `routers`, `services/context`, domain models/services, SQLC transaction conventions, initialization, Auth/OAuth, architectural acceptance, or PR #218.

## Binding decisions

1. Keep BDSPro stack: Go 1.27.3, Gin, SQLC/pgx, PostgreSQL 18, Redis/Asynq. Gitea Chi/XORM are design references, NOT requested replacements.
2. Read exact Gitea source and BDSPro source; never infer from directory names or earlier assistant commentary.
3. Canonical package ownership: `modules/web/handler.go` HTTP middleware/typed handler contract, `modules/web/router.go` Gin matching/boundary, `services/context` one request context and HTTP response, `routers/api/v1` resource actions, `services` workflows, `models` domain/persistence, `models/db` SQLC transaction/executor, `modules/util` generic errors.
4. Prove code shape **and** runtime semantics. Gin `Next/Abort` is not automatically equal to `net/http` `next.ServeHTTP`; preserve request context, wrapped response writer, committed response, path params and cleanup.
5. Preserve live-session and commit-time write-authority invariants, OAuth2/OIDC, public error privacy and existing REST API.
6. No duplicate public abstraction for the same responsibility, dead helpers, or code created only for superficial naming parity.
7. Never mark P01 done while `Router[T]`, `toGinHandler`, or missing true `BeforeRouting` remain. Treat all historical status data as a snapshot.

## Required workflow

1. **Refresh GitHub**: BDSPro `dev`, branch `refactor/gitea-web-ownership-20261110`, draft PR #218, latest SHA, diff, Actions workflow/jobs.
2. **Read**: [README.md](./README.md), [IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md), [CONTINUE_PROMPT.md](./CONTINUE_PROMPT.md).
3. **Compare** matching files from Gitea `main` and BDSPro branch. Pin source blob SHAs when documenting evidence. Trace imports, receivers, function signatures, middleware order, contextual data, error/Tx ownership and actual consumers.
4. **Patch** one coherent dependency chain, not one detached function name. Run formatting and relevant tests; use GitHub Actions if local required toolchain unavailable.
5. **Regressions**: inspect job logs; fix source/test setup faithfully. Do not delete failing tests or mask hidden context creation.
6. **Report**: checkpoint, actual before/after snippets, file paths, commit, branch, CI run URL/conclusion, known differences and next blockers.
7. **Update** status document when major behavior/phase changes; keep PR draft until final acceptance; no merge to `dev` prematurely.

## Evidence template

| Evidence | Required |
|---|---|
| Gitea file/line/function | Exact link + signature + why |
| BDSPro before | Exact source + discrepancy |
| BDSPro after | Commit/PR link + actual code |
| Invariant | Gin/SQLC/API/Auth/Session preserved |
| Test | Name, run status, logs if failure |
| Remaining | Named Pxx blocker and target |
| Merge | No unless all gates pass |

## Resume anchor

Current known checkpoint at document creation: **P01 IN PROGRESS**. Read the latest working source and CI. Immediate goal: canonical `wrapMiddlewareAndHandler`, non-generic `Router`, true `BeforeRouting`, removal of legacy typed/Gin action compilation, then full tests. Subsequent P02–P07 are detailed in [IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md).

Reference sources: [Gitea handler](https://github.com/go-gitea/gitea/blob/main/modules/web/handler.go), [Gitea router](https://github.com/go-gitea/gitea/blob/main/modules/web/router.go), [Gitea APIContext](https://github.com/go-gitea/gitea/blob/main/services/context/api.go).
