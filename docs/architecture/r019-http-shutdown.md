# R01.9c — HTTP listener, readiness and shutdown ownership

## Problem and source

At `dev@07165ddc`, `cmd/serveHTTP` already owns the TCP listener and graceful shutdown. A context canceled before startup could still lead to a listener and readiness event. More importantly, when `http.Server.Shutdown` timed out on an active request, `serveHTTP` returned an error without calling `http.Server.Close`. A deferred listener close does not terminate existing active connections.

Go's API specifies that `Shutdown` waits for connections to drain until its context expires; `Close` forcibly closes active HTTP connections. Neither guarantees termination of code in handlers that ignore their request context, and neither handles hijacked connections such as WebSockets.

Sources:
- https://pkg.go.dev/net/http@go1.27.3#Server.Shutdown
- https://pkg.go.dev/net/http@go1.27.3#Server.Close
- https://github.com/go-gitea/gitea/blob/main/cmd/web.go (compare lifecycle intent, not package imports)

## Ownership and behavior

`cmd/serveHTTP` owns binding, readiness, `net/http.Server`, shutdown budget and error propagation. The router still produces only a `net/http.Handler`; no framework or global lifecycle registry is added.

- **Startup rejected:** if the command context is canceled before listener creation, return `context.Canceled` through a wrapped error; do not signal readiness.
- **Normal stop:** `Server.Shutdown` drains accepted requests, `Serve` ends and the CLI receives `nil`.
- **Timeout stop:** after 10 seconds of incomplete drain, call `Server.Close` to terminate active connections, wait for the `Serve` result and report the original deadline error. Closing active connections is a last resort, not a successful graceful shutdown.
- **Side effects:** database/Redis ownership remains in `cmd/runWeb`; their defers run after `serveHTTP` returns.
- **Non-goals:** no API changes, no new lifecycle package, no change to worker ownership, no guarantee for hijacked connections, uncooperative handler goroutines or process crash/SIGKILL.

## Falsifiable evidence

Existing `TestServeHTTPStartupAndShutdown` and `TestServeHTTPInvalidListener` cover normal response and bind failure. New tests:

- `TestServeHTTPPreCanceledContextNeverReportsReadiness`: no readiness on already-canceled command.
- `TestShutdownHTTPServerForcesActiveConnectionCloseOnTimeout`: a real TCP request blocked in a handler cannot drain; a short test deadline must produce `context.DeadlineExceeded`, force the client connection closed and allow `Serve` to exit.

The timeout is injected into the same shutdown function used by production (production budget remains 10 seconds); no global timeout mutation is used. A 401/204 response, a green unit test, and a successful CI run are evidence only for the cases exercised. Review remaining startup cancellation windows, termination of background goroutines and worker shutdown separately before closing R01.9.
