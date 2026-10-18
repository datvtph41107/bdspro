# Lecture #53 — HTTP Logging Middleware

## Source contract
The HTTP boundary uses Gin `New` with explicit JSON structured request
logging via standard library `log/slog`. Gin's default logger is disabled
because it includes raw paths/queries; a custom recovery writer discards
panic stacks and request headers.

Every request generates a fresh API-owned 128-bit random ID, returned as
`X-Request-ID`, and logs a single `http_request` event with `request_id`,
`method`, `route` (registered Gin template), `status`,
`duration_ms`, and `panic_recovered`.

Unknown route path becomes literal `unmatched`; 500+ logs at ERROR, others
at INFO. Recovery generates status 500 but does not emit raw panic values.
The verification resend error event is logged by category and request ID,
never by raw backend error content.

## Security rule
Never write raw request URL/query, Referer, Cookie, Authorization, client IP,
request/response body, email, session, or verification tokens into generic
HTTP request logs. Do not accept arbitrary inbound X-Request-ID as trusted
correlation. A request ID is **not** an authentication identity.

Privacy tradeoff: suppressing recovery stack details makes diagnosis harder;
restore stacks only through a separate reviewed redacted/restricted channel.

## Acceptance
HTTP tests cover 200/401/403/404, registered route templating, unknown
path redaction, spoofed request ID, token body omission, and panic recovery.
CI must pass migrations UP-DOWN-UP, SQLC/source diff, tests, race tests,
startup and Docker Compose smoke. No AWS, schema, Docker config changes
or additional logging dependencies.
