# IPActive — source admission

This module follows the **action-based contract** of
[Laravel DRX IPActive](https://github.com/datvtph41107/laravel-drx/tree/main/modules/IPActive),
with Go and Redis semantics appropriate for BDSPro.

## Ownership

- `define.go`: named actions and default limits. It is the counterpart of DRX `Define.php`.
- `redis.go`: atomic Redis decision/recording, TTL and stable source key ownership.
- `event.go`: privacy-preserving admission result for optional audit and metrics.
- `middleware.go`: Gin adapter packaged with the mechanism; the caller supplies
  the application's HTTP response writer, so this module never imports routers.
- `modules/setting/ipactive.go`: optional `app.ini` policy overrides validated on startup.

A Redis client is **borrowed** from `modules/conn`; this module does not own
connection shutdown. It is not an interchangeable data cache.

## DRX behavior and backwards compatibility

DRX uses two behaviors: an interval between actions (Register, Reset, etc.)
and an attempt count (Login). Both are supported:

- `ModeCount`: fixed window beginning at the first request; atomic `INCR`
  and `EXPIRE`. Denied attempts are also counted. **All 21 existing routes
  keep this mode, same limits/windows, and their exact action identifiers.**
- `ModeCooldown`: at most one admitted request per interval; atomic
  `SET NX EX`. Denied attempts do not extend the cooldown.

The Redis namespace remains `bdspro:source_limit:v1:`; source IDs remain
SHA-256 of the trusted Gin `ClientIP()` value, preserving live counters.
Switching the mode of an existing action is an operational policy migration,
**not** something silently applied by deploying this module.

Each allowed/denied attempt is committed *inside the Redis script*. DRX's
Event -> Listener -> SQL `IpActiveLog` pipeline would introduce a race between
checking and recording; the `Event` result here is for optional observation
only. It does not authorize admission or enqueue work.

## Configuration

Policies default to the previous source-rate middleware declarations. To
override them explicitly, set the relevant `[ipactive]` fields in `app.ini`:

```ini
[ipactive]
LOGIN_LIMIT = 30
LOGIN_WINDOW = 1m
LOGIN_MODE = count
; To opt into DRX-style interval semantics for registration:
; REGISTER_MODE = cooldown
; REGISTER_LIMIT = 1
; REGISTER_WINDOW = 30s
```

Overrides may also be supplied through `BDS__IPACTIVE__LOGIN_LIMIT`, etc.
Invalid limits, durations and modes reject configuration before HTTP starts.
No action is silently disabled on Redis errors: admission fails closed (503).
Denied attempts get 429 and the remaining whole-second `Retry-After`.

No cleanup cron is needed for Redis TTL counters; do not `SCAN` / `DEL` live
keys in bulk. Keep the separate per-IP/credential `AuthThrottle` and
`RequestMonitor` policies independent of this source-IP limiter.

## Acceptance

Run `go test ./modules/ipactive ./modules/setting ./routers/...`.
With `BDS_TEST_REDIS_ADDR`, integration tests cover concurrent admission,
counting, cooldown, key compatibility, isolation, invalid source and failure
behavior. Production behavior also depends on configuring **trusted proxies**
correctly; raw `X-Forwarded-For` is never accepted as a source.
