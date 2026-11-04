# R30 — Ownership review: auth JWT, JSON v2, timeutil and util

Baseline: BDSPro dev `dd6dc3984b3f3c1590a8792dcef22d73fef1a727`.
Gitea comparison: [modules/json](https://github.com/go-gitea/gitea/tree/ab295e62e4c7/modules/json),
[modules/timeutil](https://github.com/go-gitea/gitea/tree/ab295e62e4c7/modules/timeutil),
[modules/util/error.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/util/error.go),
[services/auth](https://github.com/go-gitea/gitea/tree/ab295e62e4c7/services/auth),
[services/oauth2_provider](https://github.com/go-gitea/gitea/tree/ab295e62e4c7/services/oauth2_provider).

## 1. JSON v2: implementation must follow consumers

BDSPro production uses three operations:

| API | Verified production callers | Behavior |
| --- | --- | --- |
| `json.Marshal` | `modules/cache/string_cache.go`, `services/worker/task_send_verify_email.go`, `services/auth/oauth2_client.go` | Uses Go JSON v2; nil slice/map still encode to `null` for persisted-data parity |
| `json.Unmarshal` | `modules/cache/string_cache.go`, three worker task handlers, `services/auth/oauth2_client.go` | Strict v2: duplicate names and invalid UTF-8 rejected; callers handle errors |
| `json.NewDecoder` | `services/auth/oauth2_client.go` and `oauth2_token.go` | Stream decoding from bounded provider HTTP responses |

Gitea additionally exports a swappable `DefaultJSONHandler`, relaxed UTF-8
writer, `MarshalDeterministic`, legacy `omitempty` options, case-insensitive
decoder, `Indent`, and `jsontext.Value`. **None has a verified BDSPro
production consumer at R30.** We intentionally removed those APIs instead of
pretending a Gitea-specific workaround is an application feature. No mutable
global JSON handler remains.

Do not confuse this with a complete JSON migration: Gin continues to encode
HTTP responses and `golang-jwt/jwt` parses JWTs using its own library. This
revision does **not** claim to change those codecs. The negative behavior
tests and the original R29 persisted payload/OIDC state compatibility tests
remain authoritative. Strict JSON decoding might reject malformed old tasks
that v1 accepted; for those tasks, existing `asynq.SkipRetry` behavior remains.

## 2. JWT: remove generic token package without deleting authentication

Previously `modules/token/{maker.go,jwt_maker.go,payload.go}` implemented
BDSPro-specific HS256 access/refresh JWTs. It was not a generic auth-independent
token library. It is now `modules/auth/{token.go,jwt.go,jwt_payload.go}`:

- `auth.Maker` defines issue/verify operations; callers can inject a fake
  maker in services.
- `auth.JWTMaker` owns HS256 key size, signature and JWT validation.
- `auth.Payload` owns UUID, account ID, purpose, issuance and expiry claims.
- `services/auth` owns session issuance, renewal, revocation and account
  authorization; `routers/api/middleware` validates the access credential.
- OAuth2/OIDC integration is **client authentication**; it must not be equated
  to Gitea's full OAuth2 authorization-server/token service.

The source path changed, not the JWT claim field names, algorithm, signed
wire data or session storage. A fixed legacy-signature test proves existing
JWTs remain verifiable. There is no `modules/token` directory in R30.

## 3. timeutil: two real shared time invariants

Gitea's `timeutil.TimeStamp` and `TimeStampNano` are shaped by its data
representation and UI. BDSPro's PostgreSQL SQLC model already uses
`time.Time`; copying those types would create competing representations.

- `timeutil.SecondsCeil(duration)` avoids truncating positive TTL windows to
  Redis `EXPIRE 0`. Used by `AuthThrottle.Allow`,
  `IPActiveSource`, and `RequestMonitor`; invalid/nonpositive admission
  windows now fail closed before issuing Redis operations.
- `timeutil.Expired(now, deadline)` treats equality as expired. Used by JWT
  claim validity and PostgreSQL refresh session validation.

No process-global mock clock and no additional DB timestamp type are created.
Unit tests cover negative, zero, sub-second, exact-second and boundary cases.

## 4. util: safe error classification, not an HTTP framework

`util.WithKind` already classifies persistent domain failures used by
`models/account`, `models/auth` and `models/organization` with
`errors.Is`, without embedding HTTP status codes. Keep that behavior.

`util.WithCause(safe, cause)` adds the missing property: `Error()`
contains only a safe domain message, while `errors.Is`/`errors.As`
can still identify a database cause. Used by
`services/auth.AuthService.Register` instead of
`errors.Join(ErrEmailRegistered, pgError)`, which concatenated raw PG error
strings. HTTP status/code mapping still belongs to `routers/api/response`.

`modules/util/random.go` contains pseudo-random fixture helpers; it is not
an authentication secret generator. Production authentication proofs use
`modules/secret` and `crypto/rand`. Do not reuse `util.RandomString`
for tokens.

## Acceptance boundary

- No changes to Gin routes, SQL schema, outbox payloads or JWT wire signatures.
- Build, gofmt, SQLC and API checks on the exact head commit.
- `go test -race ./...` and dedicated JSON, JWT compatibility, TTL and safe
  error tests.
- Existing OAuth2 and authentication integration behavior must remain green.
- Merely having a new package is insufficient: each new exported function must
  have a non-test caller and be covered by a behavior test.
