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

## Optional cache configuration (R28)

The optional `[cache]` section is read only by `setting.LoadCacheSettings` when
an explicit consumer builds `modules/cache`. It does **not** replace `[redis]`
used by rate limits, OIDC state and Asynq.

```ini
[cache]
ADAPTER = memory
HOST = 127.0.0.1:11211  ; only for Memcached
PREFIX = cache:
MAX_ENTRIES = 10000
ITEM_TTL = 5m
```

`ADAPTER = redis` requires a client borrowed from `modules/connection`.
Cache errors are returned to the caller; they are not silently treated as
misses. A memory backend is per-process, not a multi-instance shared cache.
