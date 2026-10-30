# Error, Response & Logging Contract — BDSPro

**Implementation baseline:** ENG-CORE PR 1; no API success-response or public error-code migration.

## Responsibility and developer workflow

| Owner | Contract |
|---|---|
| `internal/errors` | Application failure identities; may wrap underlying causes. No `net/http` dependency, public code or client message. |
| `internal/service` | Interpret operation-specific database/provider failures; return application failures or errors with relevant context. No client JSON. |
| `internal/api/response/catalog.go` | **Single owner** of the 29 existing application error → HTTP status, stable code and safe message mappings. |
| `internal/api/response/writer.go` | Bind errors (400/413), expected application errors and safe unexpected 500 responses. |
| HTTP middleware/handlers | Map HTTP-specific failures at the ingress boundary. Existing direct `response.Reject` calls remain for a separately reviewed migration. |
| HTTP access logger | Request ID, route template, method, status, latency, and public `error_code` when available. |
| Worker/error diagnostics | Report operational outcomes without serializing task secrets or raw credentials. |

### Adding an application error

1. Define a named application error identity in `internal/errors` without `http.Status*`.
2. Classify the cause in the operation using `errors.Is`/`errors.As`; retain relevant causes with `apperrors.Wrap` when needed.
3. Add exactly one HTTP mapping in `internal/api/response/catalog.go`; choose status, stable code, safe message based on the operation's semantics.
4. Add/extend contract tests for the 4xx/5xx response, including cases where causes must not appear in client JSON.
5. Update OpenAPI response schemas when changing observable HTTP behavior; do not silently rename existing codes.
6. Keep handler usage straightforward: `if err != nil { response.Error(c, err); return }`.

### Error categories and logging

- **Expected 4xx**: normally do not log as an operational ERROR. Security denial monitoring is a separate concern.
- **Unexpected 5xx**: report a bounded safe diagnostic category, plus the request ID from the existing logger context. Never blindly serialize a provider/driver error string; it may contain credentials or tokens.
- **Dependency unavailable**: use 503 only where the capability's failure policy explicitly supports it. Do not globally map `pgx.ErrNoRows`, SQLSTATE 23505, or all network errors to a single HTTP status.
- **HTTP-only failures**: `response.Reject` still writes the existing `{code,error}` shape; the next PR must centralize repeated inline codes and add missing dependency diagnostics, without inventing a general-purpose exception framework.
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

- Inventory and unify **direct** `response.Reject` codes in Google/Auth/middleware, notably unprefixed reset/verification errors and `identity_store_unavailable`; protect any client compatibility.
- Add safe dependency diagnostics and distinguish Google authorization errors from provider outages.
- Extend OpenAPI beyond route-only coverage; verify `WWW-Authenticate` for Bearer 401.
- Verify request-to-worker correlation, log levels, outbox retries, and failure injection.

References: https://go.dev/wiki/ErrorValueFAQ and https://google.github.io/styleguide/go/best-practices.html and https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html .
