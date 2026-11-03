# Developer tools, outputs and retention

| Artifact | Produced by | Tracked? | Why |
| --- | --- | --- | --- |
| `bin/bdspro` | `make build` | No | local executable, independent of repo source |
| `.air/` | Air watcher (`.air.toml`) | No | transient rebuild and restart state |
| `models/db/*.go` | `make sqlc` | Yes | reviewed Go output from versioned SQL |
| `templates/swagger/openapi.json` | reviewed API contract | Yes | canonical OpenAPI 3 reference |
| `templates/swagger/*` UI | copied reviewed distribution | Yes | embedded static assets, licenses retained |
| `custom/conf/app.ini` | `make config-local` / developer | No | personal runtime instance and secrets |
| `data/log`, Docker log volume | modules/log and runtime setup | No | runtime diagnostics, not source |
| Docker image `bdspro:local` | Dockerfile multi-stage | No | runtime artifact |
| `tests/integration/migrations/*.sh` | humans, CI | Yes | reproducible historical schema tests |

The Makefile is a **task launcher**, not an application settings framework.
`.air.toml` is Air's build/watch/restart specification, not a backend
lifecycle implementation. `docker-compose.yaml` owns local service
topology and persistence, not application routes or domain policies.

Do not create `tools/`, `bin/`, `options/`, `scripts/` or `logs/`
source directories solely to mirror Gitea. Introduce a maintained tool
only when it eliminates a reproducible task that is not already covered
by Make, Go tooling, Docker or a package test.
