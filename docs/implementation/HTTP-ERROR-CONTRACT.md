# HTTP Request Validation & Error Contract — Engineering Core

## Source of truth

This chapter is the code-oriented follow-up to the browser security lecture.
`api/errors.go` owns the public error schema; handlers retain responsibility
for translating their domain errors to the appropriate HTTP status/message.

```json
{
  "error": {
    "code": "invalid_request",
    "message": "invalid request body",
    "request_id": "32-lowercase-hex-characters"
  }
}
```

The response code is selected from a fixed allowlist by HTTP status, not
derived from PostgreSQL/pgx messages. All error responses have
`application/json; charset=utf-8`, `Cache-Control: no-store` and the
same server-generated request ID as `X-Request-ID` and the safe HTTP log.
Unknown/unexpected errors are always `500 internal_error` and never echo
`err.Error()`. Tenant permission failures continue to collapse to 404
rather than disclosing resource existence.

| HTTP | code | Meaning |
|---|---|---|
| 400 | invalid_request | Invalid payload or domain input |
| 401 | unauthorized | No valid session |
| 403 | forbidden | Browser source/CSRF denial |
| 404 | not_found | Unknown or unauthorized resource |
| 405 | method_not_allowed | HTTP method not registered for route |
| 409 | conflict | Known domain constraint or idempotency conflict |
| 413 | payload_too_large | Body exceeded endpoint-specific limit |
| 415 | unsupported_media_type | Explicit Content-Type other than JSON |
| 500 | internal_error | Unexpected/unclassified server failure |
| 503 | service_unavailable | Authentication dependency, shutdown or readiness |

## JSON input boundary

`bindJSON` continues to use the existing Gin validator and unknown-field
rejection, with explicit early checks for media type, body size and exactly
one JSON object. Endpoint-specific caps remain 8192 bytes (Auth),
16 KiB (Organization) or 1 MiB (Source). The function writes its own
400/413/415 error response, and handlers must simply return on false.
Missing Content-Type remains temporarily accepted for legacy non-browser
clients; explicit `text/plain`, invalid media types and request bodies with
multiple top-level JSON values are rejected.

Struct validation for Organization and Source is still primarily performed
by their service/domain logic. This change does **not** claim that every
business rule can be enforced by JSON tags.

## Compatibility and security

- HTTP **statuses** of previously mapped domain failures are retained.
- Success payload structures, `204` empty bodies, cookie policies, and
  health/liveness response text remain intact.
- Error body changes from text/plain to a JSON envelope. Client developers
  must inspect `error.code` and `error.request_id` instead of comparing
  error message strings.
- Browser middleware/CSRF, malformed JSON, CORS failure, unknown routes,
  unexpected panic, DB errors, and graceful draining now use one public
  error envelope. Never expose request bodies, credentials, internal SQL or
  provider error strings in public responses.
- The response does not include stack trace or field-level details,
  intentionally minimizing user-data exposure.

## Acceptance gates

`go test ./api -run 'PublicError|JSONRequest|Recovery|SuccessfulResponses'`
plus all existing Auth, Organization, Source, RBAC and CORS tests.
GitHub Actions also checks gofmt/go vet, SQLC-generated code, migrations
UP-DOWN-UP, full race detector, server startup and Docker Compose smoke.
No AWS, schema migration, new packages or changed deployment configuration.
