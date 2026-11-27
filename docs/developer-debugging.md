# Developer debugging — BDSPro

Scope: local WSL/Linux, VS Code Go and Delve. Logging shows what happened; a breakpoint shows live state; profiling measures where resources are consumed. No runtime dependency, HTTP debug route, remote Delve listener or pprof server is added.

## 1. Prepare local resources

Open the repository root in VS Code using WSL (`code .`). Install the Go extension in WSL, check `go version` against go.mod, then install a compatible Delve using Go: Install/Update Tools or `go install github.com/go-delve/delve/cmd/dlv@latest`. Check `dlv version` and PATH. If latest does not support the project's Go version, select an explicitly compatible release; do not bypass Delve's Go-version check.

Use a disposable local database and local Redis. With Docker Compose, start only dependencies: `docker compose up -d postgres redis mailpit`. Do not start the API container on the same port. `make config-local` copies the example only if no config exists. Review the existing config before using it: development, loopback HTTP address, local DB/Redis, no live phone provider. Run `go run . --config custom/conf/app.ini config check`. Run `make migrateup` only for the intended local database if migrations are needed; never against a shared/production DB.

Set these keys in the existing sections (do not duplicate sections or replace secrets):

```ini
[log]
MODE = console,file
FORMAT = console
LEVEL = info
ROOT_PATH = data/log
[log.console]
FORMAT = console
LEVEL = info
COMPONENTS = default,http
[log.file]
FORMAT = json
LEVEL = debug
FILE_NAME = bdspro.log
[database]
LOG_SQL = all
SLOW_QUERY_THRESHOLD = 200ms
```

Keep DB_SOURCE and other required settings. The log file must be JSON and at DEBUG for the smoke test. With file DEBUG the application emission level is DEBUG, so API error stack frames are captured even though terminal is INFO.

## 2. Launch and inspect

Use F5 with "BDSPro: Web (local WSL)" from .vscode/launch.json. The working directory is the repository root, CLI args are --config ... web. The other two launch entries debug existing Go tests without a live HTTP server. Or run `make debug` in a terminal (Delve CLI). At its prompt:

```text
(dlv) break cmd.runWeb
(dlv) break github.com/datvtph41107/bdspro/routers/api/v1/auth.Login
(dlv) continue
```

For reliable file breakpoints, open the file and set a breakpoint on an executable source line in VS Code, or use the current numeric file line in Delve. The function form `break github.com/datvtph41107/bdspro/routers/api/v1/auth.Login` is an alternative. `next`, `step`, `stepout`, `stack`, `locals`, `goroutines`, `continue` inspect/advance the suspended program. Avoid printing tokens, password, raw request data, private identity information or DB credentials in shared output.

| File | Breakpoint / purpose |
| --- | --- |
| main.go | main: process entry |
| cmd/web.go | runWeb, prepareWeb, serveInstalled, listen: CLI, settings, resource lifetime |
| routers/request_log.go | requestLoggingWithThreshold: ID, activity, started, final event |
| modules/web/handler.go | fn.Call(args): adapter executes a typed action |
| routers/api/v1/api.go | Health or bind: route and form binding |
| routers/api/v1/auth/session.go | BrowserMutation / Login / writeAuthError |
| services/auth/login.go | LoginPassword, issueSession: credential checks and session creation |
| models/db/context.go | WithTx: transaction lifecycle |
| models/db/sqlc/session.sql.gen.go | CreateSession: generated SQL call |
| models/db/query_log.go | TraceQueryEnd: elapsed, SQLC name, SQLSTATE |
| services/context/api.go | APIErrorAuto / APIErrorInternal: public error boundary |

For a successful login use an existing disposable verified account through the local frontend; do not seed automatically or bypass authorization. Inspect only non-secret state such as status, handler metadata, query name, and whether an error is nil. Pausing may halt goroutines, change timeout/slow events and make HTTP clients time out. Resume before running the smoke test; never use debugger timings as performance evidence.

## 3. Acceptance: no credentials required

Start the local server with `make server` or F5; in another WSL terminal run `make smoke-logging`. The Python standard-library script restricts base URLs to literal loopback, disables proxy/redirect, sends no credentials, does not create accounts or invoke providers. Its POST deliberately fails BrowserMutation before admission and business work. It checks full server UUID in response and matching JSON events, status, severity, order, action metadata, and absence of its query/untrusted ID marker. It does not cover successful authenticated flows, provider failures, all privacy fields, or concurrent rotation; if the file rotates during a run, retry with a quiet local instance.

Expected smoke output (example, IDs vary):

```text
PASS GET /api/v1/health 200 rid=...
PASS GET /api/v1/me 401 rid=...
PASS POST /api/v1/auth/login 403 rid=...
PASS GET /api/v1/__logging_probe_missing__ 404 rid=...
PASS HTTP/log correlation (4 cases)
```

`make verify-logging` runs script unit tests and Go race-enabled tests for log/settings/web/routers/db/auth. Then `go build ./...`, `go vet ./...`, `go test ./...` to establish full baseline. The script can also be run with `make smoke-logging LOG_BASE=http://127.0.0.1:8081 LOG_FILE=data/log/bdspro.log`. Never point a forwarded loopback port at production.

## 4. Profiling without changing HTTP server

`make bench-logging` measures the same serialized HTTP event in full-text and compact format, using io.Discard. Compare ns/op, B/op and allocs/op across repeated runs. It excludes disk, network, source hook, database and actual request throughput.

`make profile-logging` records CPU/memory profiles and a test binary in ignored bin/. Inspect locally:

```sh
go tool pprof -top bin/log-profile.test bin/log-cpu.pprof
go tool pprof -top -alloc_space bin/log-profile.test bin/log-mem.pprof
```

Use an optimized build, without breakpoints, to measure performance. No pprof listener or profiler port in production. Profiles can contain internal symbols; keep them local. A profile is not an end-to-end performance claim.

## 5. Acceptance boundary

The six logging stages are source changes, not proven production behavior. Run full package tests/race/build and the smoke test on the actual Go version and dependencies. Verify real IAM success/denial, SQL failures, provider failure with a local mock, graceful shutdown and file rotation separately. Record the commit, commands, outputs, failures and tradeoffs; stop expanding logging until these are reviewed.

References: [VS Code Go debugging](https://github.com/golang/vscode-go/blob/master/docs/debugging.md), [Delve getting started](https://github.com/go-delve/delve/blob/master/Documentation/cli/getting_started.md), [Go diagnostics](https://go.dev/doc/diagnostics).
