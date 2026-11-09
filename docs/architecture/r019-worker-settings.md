## R01.9b — Worker settings and startup dependency contract

`cmd worker` is not an HTTP server. Worker responsibilities: PostgreSQL outbox/retention, Redis/Asynq, SMTP delivery, and the symmetric key for team-invitation tokens. The previous `loadSettings()` required listener, migration URL, HTTP JWT durations and OIDC settings that the worker did not use.

Now `modules/setting/worker.go` loads worker-only settings from the same `app.ini` provider, while:
- `modules/setting/audit.go::LoadAuditRetentionDays` owns one retention rule for web, worker and cleanup.
- `modules/setting/auth.go::validateTokenSigningKey` owns one signing-key rule for web and team-invite worker.
- `modules/setting/log.go::LogGlobalConfig.Validate` owns one log validation rule.
- `cmd/helper.go::loadWorkerSettings` selects and logs the config; `cmd/worker.go` only uses the exact settings it needs.
- `modules/queue`, `models/db` and `modules/conn` remain responsible for queue, SQLC transactions and Redis clients. No new process-global registry.

Runtime start/stop order remains unchanged: DB → Redis → enqueuer → processor → dispatcher + retention; shutdown cancels jobs and waits before closing queue/enqueuer, then Redis and DB. This matches the **ownership and lifecycle intent** of Gitea's `routers.InitWebInstalled` and `modules/setting` without duplicating `mustInit` fatal-on-error behavior.

No API routes, migrations, token formats, outbox task names or response semantics are changed.

Reference: https://github.com/go-gitea/gitea/blob/main/routers/init.go
