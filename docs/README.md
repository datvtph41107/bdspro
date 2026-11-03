# BDSPro developer documentation

Navigation for the **current R01 implementation**. Code and passing CI
on the exact commit take priority over explanatory claims in these files.

## Start here

- [Building the binary](build-source.md)
- [Local development and Air](development.md)
- [Testing and acceptance](testing.md)
- [Backend coding and ownership](guidelines-backend.md)
- [Current architecture](architecture/README.md)
- [Modules and real technical mechanisms](architecture/modules.md)
- [Business services](architecture/services.md)
- [HTTP routers](architecture/routers.md)
- [Gitea comparison and deliberate differences](architecture/gitea-ownership-r01.md)

## Operating the application

- [Configuration and paths](operations/configuration.md)
- [Docker and Compose](operations/docker.md)
- [Database migrations](operations/migration.md)
- [Logging](operations/logging.md)
- [CI YAML and acceptance guarantees](operations/ci.md)
- [Build tools and generated assets](operations/developer-tools.md)
- [API and Swagger](api/README.md)
- [Historical decisions and lecture material](archive/README.md)

## Source ownership at a glance

| Concern | Source |
| --- | --- |
| Command/process | `main.go`, `cmd/` |
| Runtime config/log | `modules/setting/`, `modules/log/` |
| Queue transport | `modules/queue/` |
| HTTP API and protocol | `routers/api/`, `routers/common/` |
| Business processes | `services/auth/`, `services/organization/`, `services/jobs/` |
| Worker policy | `services/worker/` |
| Database/SQL | `models/db/`, `db/query/`, `db/migration/` |
| Runtime assets | `templates/auth/`, `templates/swagger/` |
| Developer workflow | `Makefile`, `.air.toml`, `docker-compose.yaml` |
| Historical migration fixtures | `tests/integration/migrations/` |

Gitea's docs similarly distinguish build, development, testing and backend
guidelines. This navigation adopts that **workflow** without claiming
BDSPro has Gitea's Git repository, issue, package or web frontend systems.
