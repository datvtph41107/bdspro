# Configuration: developer, Docker and deployment

## What `custom/conf` means

Gitea itself retains `custom/conf/app.example.ini` and resolves a
`custom/conf/app.ini` instance file by default. BDSPro therefore keeps
this convention for **local** development; it is not a mandatory runtime
path in every environment.

| Location | Owner | Meaning |
| --- | --- | --- |
| `custom/conf/app.example.ini` | Git | committed local sample only |
| `custom/conf/app.ini` | developer | ignored instance settings |
| `docker/root/etc/templates/app.ini` | image build | immutable first-boot template |
| `/data/bdspro/conf/app.ini` | container volume | persisted runtime settings |
| `modules/setting/path.go` | application | resolve selected config path |
| `modules/setting/provider.go` | application | read INI plus explicit env overrides |
| `modules/setting/config.go` | application | typed settings and validation |

`make config-local` creates the ignored local INI only when missing.
`make server`, `make worker` and `make migrateup` pass the same
`CONFIG` argument. No service URL belongs inside the Makefile.

## Effective config precedence

- Root `--work-path` overrides `BDS_WORK_DIR` (which overrides current
  shell working directory) as the application working directory.
- Default INI: `<work-path>/custom/conf/app.ini`.
- Explicit `--config` always selects its own file. Relative paths are
  resolved against the **shell current working directory**.
- `BDS__SECTION__KEY` overrides an INI value **in memory**. Default
  section has `BDS__RUN_MODE`.
- These in-memory overrides do *not* persist. Gitea Docker
  `environment-to-ini` instead writes into persisted config; do not
  describe these as the same mechanism.

Examples:

```sh
make config-local
make server CONFIG=/absolute/path/to/app.ini
./bin/bdspro --work-path /absolute/instance web
BDS__DATABASE__DB_SOURCE='postgresql://...' make migrateup
```

`--help` and `--version` must succeed without config or database.
When debugging, log the resolved file path but never database passwords,
client secrets, JWT signing keys, authorization headers or OTPs.

**Security:** examples contain development-only credentials and signing
keys. Do not deploy examples unchanged. The Docker Compose file is a
development environment, not production hardening.

## OAuth2 config parity across local and Docker instances

Both committed example INI files use the current `[oauth2]` settings:
`ISSUER`, `AUTHORIZATION_URL`, `TOKEN_URL`, `JWKS_URL`,
`CLIENT_ID`, `CLIENT_SECRET`, and `REDIRECT_URI`.
The previous Docker-only `[oauth] GOOGLE_*` example is obsolete.
`modules/setting` continues to reject **non-empty** legacy OAuth provider
keys rather than silently guessing their meaning.

The Docker entrypoint copies the image template into
`/data/bdspro/conf/app.ini` **only if it does not already exist**. Updating
the image does not rewrite a persisted instance configuration. Operators
must migrate any populated legacy keys to `[oauth2]` in the existing
volume, preserving actual credentials securely; do not delete the app.ini
volume or overwrite it with example secrets.

## Worker queue and log configuration (post-P06)

The **optional worker-only** `[queue]` section controls Asynq concurrency and priority weights. Missing keys use the existing defaults 10/10/5; explicit empty, invalid or zero values **fail worker initialization before opening PostgreSQL or Redis**. `web`, `migrate`, `help` do not require the worker queue section.

```ini
; Optional worker-specific queue configuration
[queue]
CONCURRENCY = 10
CRITICAL_WEIGHT = 10
DEFAULT_WEIGHT = 5
```

BDSPro bounds concurrency at 256, each queue weight at 100 and total queue weight at 1000; these are application operational safeguards, not Asynq/Gitea library restrictions. Queue names `critical` and `default` must match durable outbox task producers. The deployed worker uses Asynq + PostgreSQL outbox, not an in-process channel queue.

The configured `[log] ROOT_PATH` is resolved relative to the canonical `--work-path` when not absolute. Only selecting a file logger creates the directory (mode 0700) and file (0600); an unusable output fails startup rather than silently falling back to console. `main` closes the process log sink.

**Removed configuration:** P06 removed the unused `modules/cache/**`, `modules/setting/cache.go` and channel task queue backends. Do not set a `[cache]` stanza in current templates or rely on old `LoadCacheSettings`/Memcached constructors. Redis remains mandatory for web authentication throttling and Asynq worker preflight, independent of generic caching. Existing persisted Docker `app.ini` is **not** overwritten by updating the image template; operators should inspect it for deprecated keys before rollout.

See [current queue/Redis ownership](../architecture/cache-queue.md) and the [Chat C P06 handoff](https://github.com/datvtph41107/bdspro/pull/220#issuecomment-6764418034). Final behavior is verified only on the P07 combined HEAD and PostgreSQL18/Redis/HTTP runtime tests.
