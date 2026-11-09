# Backend code ownership guidelines

Gitea's architecture is a working system with different transports and
domain operations; **folder names are not a pattern by themselves**.

| Directory | Owns | Must not own |
| --- | --- | --- |
| `cmd/` | CLI, process startup and shutdown | HTTP endpoint policy, domain rule |
| `modules/setting/` | path resolution, INI decoding, config validation | DB connection |
| `modules/log/` | writer/format/level lifecycle | domain event decisions |
| `modules/conn/` | Redis client connection lifecycle | security business policy |
| `routers/common/` | cross-transport HTTP protocol | duplicate Gin router |
| `routers/api/v1/` | Gin API route registration and resource HTTP handlers (no URL prefix change) | SQL transactions |
| `services/auth/` | credentials, sessions, phone and OAuth behavior | Gin responses |
| `models/organization/` | membership and organization persistence invariants | HTTP/transport responses |
| `services/worker/` | Asynq task execution and outbox delivery | synchronous request processing |
| `models/db/` | SQLC queries, pgx connection, SQL transaction machinery | HTTP or job transport |
| `templates/` | embedded HTML and Swagger assets | persistent data models |

When adding a feature: find the API route → request binding → model or business
operation → SQL query → migration. Keep names concrete, use Gin as the only HTTP engine, the existing typed modules/web adapter for new
API actions, and SQLC rather than a second middleware engine or generic repository registry.

`models/db` is a persistence implementation, not a second domain service.
The `models/repo`, `models/issues`, `models/secret`,
`services/actions`, `routers/web`, `routers/private`, `options` and
`web_src` counterparts in Gitea arise from real Git repository, issue,
secret, internal-HTTP, template and frontend responsibilities. Do not add
empty equivalents for future intentions.

A `GORM` adoption needs a separate proof of query/migration ownership;
SQLC and versioned SQL remain canonical in R01.

Read [ownership evidence](architecture/gitea-ownership-r01.md) and the
[P0 programming-model inventory](architecture/gitea-programming-model-p0.md).
New routes must not increase the existing Gin registration/Wrap migration debt;
see tests/architecture/programming_model_test.go for enforceable limits.
