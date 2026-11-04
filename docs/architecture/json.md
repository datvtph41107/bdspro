# R29 — JSON serialization policy

Baseline: Gitea `ab295e62e4c7`; BDSPro `dev` after R28.
References: [Gitea json.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/json/json.go),
[Gitea jsonv2.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/json/jsonv2.go),
[Go 1.27 JSON v2 release notes](https://go.dev/doc/go1.27#encoding/json/v2).

## Why the module exists

This is a **serialization policy boundary**, not a DTO framework or a wrapper
for its own sake. Gitea's `modules/json` picks a single JSON engine, offers a
stable marshal/stream API, and exposes deliberate exceptions to normal encoding.
BDSPro uses Go 1.27.3 `encoding/json/v2`, preserving nil slice/map as `null`
and keeping v2's **strict** unmarshal defaults (including duplicate-key and
invalid-UTF-8 rejection). A future engine switch must satisfy the same behavior
tests, not just compile.

| File | Owns |
| --- | --- |
| `modules/json/json.go` | public `Interface`, `Encoder`, `Decoder`; `Marshal`, `Unmarshal`, `NewEncoder`, `NewDecoder`, `Indent`, `MarshalIndent`, `Valid`, `MarshalWriteRelaxed` |
| `modules/json/jsonv2.go` | Go 1.27 v2 engine, strict/default options, relaxed invalid-UTF-8 writer, deterministic serialization, legacy optional-empty handling, case-insensitive opt-in, `jsontext.Value` |
| `modules/json/json_test.go` | semantics: nil collection, duplicate member rejection, UTF-8, streaming, formatting, case matching and deterministic output |

`json.Valid` performs **syntax-only** validation using the v1 helper; it does
not mean that `Unmarshal` will accept duplicate names or a value suitable for
the requested Go type.

## Applied operations, with existing contracts retained

| Existing owner | Usage | Compatibility and risk |
| --- | --- | --- |
| `modules/cache/string_cache.go` | `PutJSON` / `GetJSON` | JSON v2 round-trip; a backend outage remains an error, not a cache miss |
| `services/worker/task_send_verify_email.go` | serialize and parse persistent Asynq payloads | task type and `account_id` JSON key remain unchanged |
| `services/worker/task_send_password_reset.go` | parse existing queued jobs | JSON v2 stricter parsing; duplicate members are rejected rather than ambiguously accepted |
| `services/worker/task_send_team_invite.go` | parse invitation IDs | keeps `invite_id` UUID representation and SkipRetry behavior |
| `services/auth/oauth2_client.go` | encode/decode single-use state; decode OIDC token endpoint response | strict JSON for state and provider response; Redis GETDEL ownership unchanged |
| `services/auth/oauth2_token.go` | decode JWKS over bounded HTTP response | strict JSON; cryptographic signature and claims verification still belong to the JWT/OIDC verifier |

## Important non-goals and test rules

- **Gin owns HTTP JSON response serialization** in this PR. Replacing Gin's
  writer would be a separate, contract-sensitive change; endpoint responses
  must not silently change omitempty or array/null semantics.
- `golang-jwt/jwt` still parses JWT claims using its own implementation.
  A `modules/json` decoder does **not** establish strictness throughout
  that library. The JWT verifier must continue checking algorithm, issuer,
  audience, signature, time and nonce.
- `MarshalDeterministic` is useful for repeatable outputs; it is **not**
  a canonical JSON signature standard. Never sign JSON by assuming it is.
- `MarshalKeepOptionalEmpty` and `NewDecoderCaseInsensitive` are explicit
  compatibility exceptions. Do not use case-insensitive decoding in sensitive
  OIDC or task payload paths.
- `MarshalWriteRelaxed` replaces malformed UTF-8 rather than returning an
  error; it should not be used for security proofs.
- No package may mutate `DefaultJSONHandler` after startup.
- JSON v2 rejects some payloads that v1 accepted. The worker's malformed
  payloads retain `asynq.SkipRetry`; document and test any legacy payload
  formats before a larger migration.
- Model invariants, domain validation, cache TTL and queue retry policy
  belong to their respective packages, not this JSON codec.

## Acceptance

Run `go test ./modules/json ./modules/cache ./services/auth ./services/worker`,
then the existing GitHub Actions pipeline with Go 1.27.3 (SQLC, migration,
`gofmt`, race test, Docker, HTTP smoke). Test old-v1-encoded Asynq payloads
against the new decoder. Do not merge PR #121 on a green check for a prior SHA.
