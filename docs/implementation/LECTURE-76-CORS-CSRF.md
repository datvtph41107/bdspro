# Lecture #76 — Browser Security: CORS and CSRF

## Why

Previously `checkOrigin` only rejected an explicit foreign `Origin` or
`Sec-Fetch-Site: cross-site`; it omitted real preflight CORS responses and
allowed cookie-authenticated mutations with neither browser provenance
header. `SameSite=Lax` protects some browser cross-site traffic, but it
is not a full CSRF policy and does not distinguish sibling origins.

## Contract

`api.browserSecurity` is an HTTP middleware, **not** authentication
(`requireSession`) or tenant authorization (Service/SQL). It runs after
the request ID/logger and shutdown admission check.

1. For `Origin` matching configured `ALLOWED_ORIGIN` **or the API's own
   actual origin** (using direct Host/TLS, not untrusted forwarded headers):
   respond `Access-Control-Allow-Origin` with that exact origin,
   `Access-Control-Allow-Credentials: true`, and
   `Access-Control-Expose-Headers: X-Request-ID`.
2. All responses vary on Origin. A preflight (`OPTIONS` with
   `Access-Control-Request-Method`) is accepted only when the origin,
   method (GET/HEAD/POST/DELETE/OPTIONS) and every requested header
   (Content-Type, Idempotency-Key, X-BDSPro-Request) are allowlisted. Accepted
   preflight gets 204 and max-age 300; rejected gets 403. No wildcard
   when credentials are allowed, no Authorization-header allowance.
3. Unsafe requests (POST/PUT/PATCH/DELETE etc.) deny explicit
   `Sec-Fetch-Site: cross-site` or an untrusted `Origin`, independently
   of CORS. A cookie-bearing unsafe request, and any login/logout request,
   must also provide a trusted `Origin`, `Sec-Fetch-Site: same-origin`,
   or `X-BDSPro-Request: 1`. If all three are absent, return 403
   before invoking Auth/Organization/Source services.
4. The custom header is deliberately a non-simple browser header. A
   cross-origin JavaScript caller cannot send it without a successful CORS
   preflight. Non-browser/CLI clients can set the header directly; they still
   need a real session to access protected resources.
5. Token-free public registration/verification calls without browser
   metadata retain their API behavior for existing non-browser clients.
   This is a scoped compatibility choice, not full CSRF-token protection.

## Browser and local examples

For an approved SPA origin, browser requests should send credentials and,
when there is no useful Origin/Fetch metadata, the custom header:

```js
fetch("https://api.example.test/auth/logout", {
  method: "POST",
  credentials: "include",
  headers: { "X-BDSPro-Request": "1" },
})
```

If app and API live on different **sites**, the existing
`SameSite=Lax` cookie will not accompany cross-site fetches, and the
Fetch-Metadata policy rejects cross-site unsafe writes. That topology is
**not** supported by the current browser cookie contract. Use same-origin
or same-site deployments; do not relax cookies merely to make CORS pass.

Local CLI/HTTP test clients using a session cookie and no Origin may set
`X-BDSPro-Request: 1`, while real browsers can use same-origin metadata.

## Configuration

`ALLOWED_ORIGIN` is one exact origin with HTTP(S) scheme, host and
optional port, without path/query/fragment/wildcard or credentials.
Non-development environment requires HTTPS. Development may leave it
empty: same-origin requests still work based on request Host/TLS.
This config remains one source of truth; no extra middleware allowlist.

## Explicit limitations

- This is browser-facing CSRF mitigation, **not** per-session synchronizer
  CSRF tokens. A compromised same-origin application (XSS/client-side CSRF)
  defeats this boundary.
- The fallback custom header is not secret; native clients may spoof it.
  Its browser protection arises from non-simple CORS preflight rules.
- `Sec-Fetch-Site: cross-site` is rejected even when an Origin was listed,
  matching current Lax-cookie same-site deployment assumptions.
- These controls do not fix unsafe GET handlers, open redirects, XSS,
  production reverse-proxy trust, or rate limiting. Avoid mutating via GET.

## Acceptance

Test allowed/denied CORS preflight, credentials/no wildcard, unknown origins,
Vary/cache isolation, cookie and login CSRF reject/accept outcomes, no
service calls on rejected requests, and config-origin validation. CI must
pass gofmt, SQLC, migration UP-DOWN-UP, unit/integration tests, race, server
and Docker Compose smoke. No AWS, new auth scheme, new DB schema, or
new third-party dependency.
