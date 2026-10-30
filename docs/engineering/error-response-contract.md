# Error, Response & Logging Contract — BDSPro

**Implementation baseline:** ENG-CORE PR 1 + stacked PR 2; no success-response or public error-code renaming. HTTP-only failure contracts are now named by the response package.

## Responsibility and developer workflow

| Owner | Contract |
|---|---|
| `internal/errors` | Application failure identities; may wrap underlying causes. No `net/http` dependency, public code or client message. |
| `internal/service` | Interpret operation-specific database/provider failures; return application failures or errors with relevant context. No client JSON. |
| `internal/api/response/catalog.go` | Mapping owner for the 29 existing application errors → HTTP status, stable code and safe message; application errors are separate from endpoint-local HTTP decisions. |
| `internal/api/response/writer.go` | Bind errors (400/413), expected application errors and safe unexpected 500 responses. |
| `internal/api/response/writer.go` | **One JSON writer**; shared Bearer, rate-limit and safe 500 protocol helpers. HTTP-only operation errors remain at their HTTP caller with explicit status/code/message. |
| HTTP middleware/handlers | Call `response.Reject(c, status, code, message)` for endpoint-specific HTTP failures; use specialized response helpers when they encapsulate shared protocol behavior. |
| HTTP access logger | Request ID, route template, method, status, latency, and public `error_code` when available. |
| Worker/error diagnostics | Report operational outcomes without serializing task secrets or raw credentials. |

### Adding an application error

1. Define a named application error identity in `internal/errors` without `http.Status*`.
2. Classify the cause in the operation using `errors.Is`/`errors.As`; retain relevant causes with `apperrors.Wrap` when needed.
3. Add exactly one HTTP mapping in `internal/api/response/catalog.go`; choose status, stable code, safe message based on the operation's semantics.
4. Add/extend contract tests for the 4xx/5xx response, including cases where causes must not appear in client JSON.
5. Update OpenAPI response schemas when changing observable HTTP behavior; do not silently rename existing codes.
6. Keep handler usage straightforward: `if err != nil { response.Error(c, err); return }`.

### Adding an HTTP-specific failure

For an operation-specific HTTP decision, write it at the HTTP boundary:
```go
response.Reject(c, http.StatusBadRequest, "google_invalid_state", "invalid Google authorization state")
return
```

Only use a shared helper for a repeated rule with real protocol semantics:
```go
response.InvalidAccessToken(c)                // includes WWW-Authenticate
response.RateLimited(c, retryAfterSeconds)    // includes Retry-After
response.RateLimitUnavailable(c)              // fail-closed admission
```

Do not add a new global enum or a central `switch` for each new operation-level HTTP failure. Document and test public codes, status, messages and headers where the endpoint contract is owned. No silent renames.

### Error categories and logging

- **Expected 4xx**: normally do not log as an operational ERROR. Security denial monitoring is a separate concern.
- **Unexpected 5xx**: report a bounded safe diagnostic category, plus the request ID from the existing logger context. Never blindly serialize a provider/driver error string; it may contain credentials or tokens.
- **Dependency unavailable**: use 503 only where the capability's failure policy explicitly supports it. Do not globally map `pgx.ErrNoRows`, SQLSTATE 23505, or all network errors to a single HTTP status.
- **HTTP-only failures**: use `response.Reject` at the HTTP decision point; reserve helpers for Bearer, rate limiting and safe internal errors. All response JSON still flows through the same writer.
- **Bearer 401**: use the shared helpers that set `WWW-Authenticate` (missing token, invalid access token, or identity no longer valid). OAuth grant errors are not Bearer challenges.
- **Rate limiting**: a missing Redis client must fail closed for authentication source and identity limits. Retain the existing `Retry-After` header when rejecting with 429.
- **OAuth provider**: a rejected grant remains 401 `google_authorization_failed`; transport failures, 429, 5xx and malformed token responses become 503 `google_unavailable`, with a safe diagnostic event.
- **Client contract**: preserve `{code,error}`, HTTP status, and `X-Request-ID`. Client logic must not parse the human-readable `error` message.
- **Diagnostic limits**: safe type/PostgreSQL SQLSTATE can be logged in the first PR, but deeper cause classification and provider-specific events still need acceptance tests.

### Operational logging conventions (PR 03)

**Format and level are independent settings.** `LOG_FORMAT=json|console` controls output rendering; `LOG_LEVEL=debug|info|warn|error` controls application Zerolog events. The default `LOG_LEVEL=info`; invalid values reject configuration before starting dependencies. `LOG_LEVEL=debug` is not permission to log credentials, OTP, authorization headers, provider payloads or raw URLs.

| Event | Default level | Owner | Required safe fields |
|---|---|---|---|
| `http.request` | INFO, including HTTP 5xx | HTTP access middleware | `request_id`, method, **route template**, status, duration, public `error_code` when present |
| `http.application_error` / `http.panic` | ERROR | HTTP exception/response boundary | request ID, bounded error type; do **not** serialize raw error text |
| `auth.login_completed` | INFO on success/5xx, WARN on auth failure | Login audit | outcome/status; the operation records the actual 5xx diagnostic |
| `security.request_denied` | WARN for 401/403/429, DEBUG for routine 400/413 | Security monitor | sanitized reason, route template, method, hashed source |
| `security.suspicious_activity` | WARN | Security monitor | source hash, count, window |
| `task.failed` | ERROR | Asynq error handler | task type, task ID, retry count, maximum retries, skip-retry flag, bounded error type |
| `verification_email.completed` | INFO | Asynq task handler | task type and task ID, **not** account ID or payload |
| `queue.outbox_dispatch_failed` | ERROR | Outbox dispatcher | bounded error type, never task payload or raw DSN |

**One failure, two useful records:** a failed request can have one ERROR diagnostic **and** one INFO `http.request` outcome; do not make the access outcome a second ERROR. Do not log-and-return an error from every service call. Use wrapped errors to preserve operation context in Go, then emit diagnostics only at the boundary with sufficient context.

**Correlation boundary:** `request_id` is generated for HTTP requests; `task_id` identifies persisted Asynq jobs. These are **not automatically the same trace**, because outbox records are committed independently. This PR does not claim HTTP-to-worker trace propagation; it requires a separate data contract if later added.

**Worker caveat:** Asynq internal `Logger.Fatal` currently has process-exit semantics. Changing that behavior or the fixed Asynq Warn-level filter requires a lifecycle review and failure tests, and is not part of this PR.

**Developer workflow:** set `LOG_FORMAT=console LOG_LEVEL=debug` for local diagnostics when appropriate, or keep JSON at `info` in CI. Search logs by `request_id` and `error_code` for HTTP, or `task_id` and `task` for workers. A `500` with no bounded diagnostic, or private values in any event, fails acceptance.

### Invariants for review

- Neither application errors nor generated SQLC records are automatically public HTTP responses.
- Expected failures must preserve established status/code/message until a separately approved migration.
- A sensitive raw error string must not appear in HTTP JSON or unexpected-error logs.
- Application code must not log-and-return the same failure at every layer.
- The response mapper owns no retry, database, authorization, or provider logic.
- `TestAllApplicationFailuresKeepTheirPublicContract` captures all 29 application failure mappings.
- CI and local tests must distinguish PASS, SKIP and NOT RUN.

### Follow-up scope (not part of this PR)

- Extend OpenAPI beyond route-only coverage, including status schemas and `WWW-Authenticate` expectations.
- Decide whether historical unprefixed codes should be migrated in a versioned API contract; do not silently rename them.
- Verify request-to-worker correlation, log levels, outbox retries, and failure injection.

References: https://go.dev/wiki/ErrorValueFAQ and https://google.github.io/styleguide/go/best-practices.html and https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html .
