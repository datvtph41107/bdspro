# Main bootstrap — align with SimpleBank Lecture #12

Reference: [SimpleBank main.go at Lecture #12](https://github.com/techschool/simplebank/blob/6354a89f06f4c6b07760620fe11ff6cb2acc3cd6/main.go).

## Rules

- `main.go` owns only composition: `LoadConfig` → PostgreSQL pool → Store/Services → Gin Server → `Start`.
- `util/config.go` owns Viper and configuration validation. No environment-dependent `if/else` in `main.go`.
- `auth.NewSMTPMailer` returns `nil` only when SMTP address is empty. The service continues its existing behavior when local development intentionally has no mailer.
- `api.Server.Start` owns the existing HTTP listener, timeouts, SIGINT/SIGTERM and graceful shutdown. The server still logs `BDSPro listening` for CI.
- `Makefile` remains `server: go run .`, as in SimpleBank; no shell export wrappers.
- No changes to SQL schema, migrations, API routes, authentication cookies or authorization.

## Deliberate differences from SimpleBank

- PostgreSQL uses `pgxpool` rather than `database/sql`.
- Production cannot start without SMTP and ALLOWED_ORIGIN; development tokens cannot be served from a non-loopback listener. This preserves existing security rules, but keeps them outside the entrypoint.
- We preserve graceful shutdown and HTTP server timeouts rather than reverting to plain `gin.Run`.
- `util.LoadConfig` instantiates Viper per call; config tests verify file defaults, environment precedence, missing file and production/development safety.

## Verification

```sh
go test ./util ./auth -count=1 -v
make test
make server
```

Database tests require their explicit disposable database environment variables, as described in `ci.yml`. A local `go test ./...` may skip DB-dependent cases if these variables are absent.

Next checkpoint: SimpleBank Lecture #13 — introduce an interface and mock tests at the boundary used by the real Gin handler, without creating bank-specific Account endpoints.
