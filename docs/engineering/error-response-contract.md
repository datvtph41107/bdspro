# Error, Response & Logging Contract — BDSPro

**Implementation baseline:** ENG-CORE PR 1 + stacked PR 2; no success-response or public error-code renaming. HTTP-only failure contracts are now named by the response package.

## Responsibility and developer workflow

| Owner | Contract |
|---|---|
| `internal/errors` | Application failure identities; may wrap underlying causes. No `net/http` dependency, public code or client message. |
| `internal/service` | Interpret operation-specific database/provider failures; return application failures or errors with relevant context. No client JSON. |
| `internal/api/response/catalog.go` | **Single owner** of the 29 existing application error → HTTP status, stable code and safe message mappings. |
| `internal/api/response/writer.go` | Bind errors (400/413), expected application errors and safe unexpected 500 responses. |
| `internal/api/response/failures.go` | **Single owner** of the 22 named HTTP-only failures, including their public status/code/message and Bearer challenges. |
| HTTP middleware/handlers | Choose a named `response.HTTPFailure` when the HTTP boundary has already decided an HTTP-specific failure. Never repeat status/code/message literals. |
| HTTP access logger | Request ID, route template, method, status, latency, and public `error_code` when available. |
| Worker/error diagnostics | Report operational outcomes without serializing task secrets or raw credentials. |

### Adding an application error

1. Define a named application error identity in `internal/errors` without `http.Status*`.
2. Classify the cause in the operation using `errors.Is`/`errors.As`; retain relevant causes with `apperrors.Wrap` when needed.
3. Add exactly one HTTP mapping in `internal/api/response/catalog.go`; choose status, stable code, safe message based on the operation's semantics.
4. Add/extend contract tests for the 4xx/5xx response, including cases where causes must not appear in client JSON.
5. Update OpenAPI response schemas when changing observable HTTP behavior; do not silently rename existing codes.
6. Keep handler usage straightforward: `if err != nil { response.Error(c, err); return }`.

### Adding an HTTP-only failure

1. Add a named `HTTPFailure` and its single status/code/message mapping in `internal/api/response/failures.go`.
2. Call `response.Fail(c, response.YourFailure)` from middleware or an HTTP handler. Do not pass arbitrary public code/message literals.
3. Add a row to `TestHTTPFailureContracts` and relevant behavior tests (authentication, rate limits, OAuth, etc.).
4. Decide whether a response needs protocol headers (`WWW-Authenticate`, `Retry-After`) and ensure caller or response owns the value.
5. Preserve client-visible behavior. Similar-looking application and page errors can legitimately have different historical codes until a versioned API migration.

Example:

```go
if err != nil {
    response.Error(c, err)
    return
}

response.Fail(c, response.InvalidAccessToken)
return
```

### Error categories and logging

- **Expected 4xx**: normally do not log as an operational ERROR. Security denial monitoring is a separate concern.
- **Unexpected 5xx**: report a bounded safe diagnostic category, plus the request ID from the existing logger context. Never blindly serialize a provider/driver error string; it may contain credentials or tokens.
- **Dependency unavailable**: use 503 only where the capability's failure policy explicitly supports it. Do not globally map `pgx.ErrNoRows`, SQLSTATE 23505, or all network errors to a single HTTP status.
- **HTTP-only failures**: call `response.Fail(c, response.BearerRequired)` or the appropriate named failure. The low-level JSON writer is private; client-facing triplets are defined only in `internal/api/response`.
- **Bearer 401**: use the named HTTP failures that set `WWW-Authenticate` (missing token, invalid access token, or identity no longer valid). OAuth grant errors are not Bearer challenges.
- **Rate limiting**: a missing Redis client must fail closed for authentication source and identity limits. Retain the existing `Retry-After` header when rejecting with 429.
- **OAuth provider**: a rejected grant remains 401 `google_authorization_failed`; transport failures, 429, 5xx and malformed token responses become 503 `google_unavailable`, with a safe diagnostic event.
- **Client contract**: preserve `{code,error}`, HTTP status, and `X-Request-ID`. Client logic must not parse the human-readable `error` message.
- **Diagnostic limits**: safe type/PostgreSQL SQLSTATE can be logged in the first PR, but deeper cause classification and provider-specific events still need acceptance tests.

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
