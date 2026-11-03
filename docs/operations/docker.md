# Docker and Compose: repeatable development environment

The Docker layer has two jobs, not an independent application framework:
compile a release artifact and prepare a writable instance filesystem.

```text
Dockerfile
  builder: Go binary (no local bin/ copied)
  runner: /app/main + /app/setup + template + migrations
docker/setup
  mkdir /data/bdspro/conf and /data/bdspro/log
  create app.ini on first boot only (atomic hard link on same volume)
  exec the supplied command
docker-compose.yaml
  postgres + redis + mailpit (development dependencies)
  api     -> /app/main --config /data/bdspro/conf/app.ini web
  worker  -> /app/main --config /data/bdspro/conf/app.ini worker
```

```sh
docker compose config --quiet
docker compose up -d --build
docker compose ps
docker compose logs -f api worker
curl -i http://127.0.0.1:8080/health
```

- PostgreSQL/Redis health checks gate both application processes.
- PostgreSQL/Redis host ports bind to loopback for local development.
- Mailpit accepts disposable test email; no real emails should be sent.
- API and worker share image and persisted app volume, but are different
  processes with distinct shutdown/scale behavior.
- A new image template does **not** overwrite an existing volume INI.
  Changing config requires an explicit, reviewed mutation.
- `docker compose down` preserves volumes; **`down -v` removes
  local data**. Do not use it casually.
- Root-owned container files, example passwords, no TLS reverse proxy or
  backup policy mean this is not a production deployment guide.

The `docker-compose.yaml` is a concrete environment inventory:
`services` names each process, `healthcheck` provides readiness gates,
`depends_on` handles process ordering, `volumes` persists state,
`command` selects the CLI mode, and `logging` limits log file growth
outside the application. It does **not** imply database migration
readiness or job E2E correctness.

Unlike Gitea's s6-based Docker distribution, BDSPro uses a single
setup entrypoint; copying Gitea's supervisor would add unnecessary
process management without a proven need.
