# Lecture #74 — Graceful Shutdown & PostgreSQL lifecycle

## Existing weakness

Prior `api.Server.Start` directly called `http.Server.Shutdown` with
a detached five-second context after SIGTERM; there were no deterministic
tests for in-flight responses, request admission, transaction completion
or shutdown timeout. `main.go` already owned the `pgxpool.Pool` and called
`defer pool.Close()`, which is the correct ownership boundary.

## Implementation

- Keep `Start(address)` as the concrete listener and OS-signal owner.
- Extract unexported `serve(ctx, listener, gracePeriod)` to exercise lifecycle
  behavior without sending real OS signals.
- On cancellation, set `draining` before shutting down HTTP.
- `/ready` and new application requests return **503** while draining;
  `/health` remains **200** while the process itself is still alive.
- Existing requests keep their original request contexts during grace,
  so HTTP responses and transactions can finish and COMMIT.
- `Shutdown` uses a **new** background context bounded by five seconds
  in production: the signal context is already canceled.
- If the grace deadline expires, call `Close` to terminate active connections
  and cancel their request contexts; return the timeout error, not false success.
- Always wait for the accept loop to exit before returning from `Start`.
  The application still closes `pgxpool` afterwards through `main.go`.

## Tests

- One request is held open, server receives cancellation, request completes
  successfully, and only then does shutdown return.
- New work gets 503 while draining; health is still alive.
- A stuck handler reaches the injected test grace deadline, resulting in a
  forced active connection close and observable context cancellation.
- An unexpected listener failure returns the original error.
- In a CI-owned PostgreSQL database, one active transaction remains leased
  during drain, commits successfully, and releases the connection before
  the server returns and the test closes the pool.

## Important limits

Graceful shutdown does **not** guarantee COMMIT for a request that exceeds
its grace period, nor cancel goroutines that ignore request contexts.
`http.Server.Close` interrupts active network connections, not arbitrary
background work. `pgxpool.Close` has no context timeout and can block while
connections remain leased by abandoned goroutines. The system has no
hijacked/WebSocket sessions or background job manager today; those would
require separate lifecycle hooks if introduced.

This lecture changes Go source and tests only: no schema change, AWS,
Kubernetes, new library, deployment, or unnecessary lifecycle interface.

## Acceptance gates

Run `go test ./api -run 'Shutdown|Serve'`, `go test -race ./api`,
repository-wide Go tests/race, SQLC source diff, migrations UP-DOWN-UP,
server startup and Docker Compose smoke in GitHub Actions. Only merge
after the branch's latest HEAD has completed with SUCCESS.
