# R16 — Google OAuth flow service ownership

Gitea reference: `services/auth`, `routers/api/v1/user`, and `routers/init.go` at `go-gitea/gitea@12b20c7`.

## Source boundary

- `services/auth/google_oauth_flow.go`: create random state/verifier/nonce, persist and atomically consume the Redis state, construct PKCE S256 authorization URL, exchange code with Google. No Gin or public HTTP response dependency.
- `routers/api/v1/user/google_start.go`: reauthentication check, cookie and redirect/JSON.
- `routers/api/v1/user/google_callback.go`: compare state cookie, map errors to existing HTTP codes, call identity/account service and return response.
- `routers/api/v1/user/google.go`: initialize a single flow capability for each handler instance.

No new provider registry, middleware DSL, SQLC/ORM path, OAuth endpoint, or Redis namespace.

## Behavioral invariants

- Redis key `google:oauth:state:` and serialized JSON keys unchanged.
- TTL five minutes; `GetDel` consumes state before token exchange; replay denied.
- PKCE code challenge SHA-256, nonce, cookie `HttpOnly`, `SameSite=Lax`, and conditional Secure remain unchanged.
- Google 429/5xx, network and malformed response => existing 503 `google_unavailable`; OAuth grant rejection => existing 401 response.
- No raw OAuth code, token, state, verifier or provider error body in application logs.

## Acceptance

Go 1.27.3 CI on PR head with `go test ./...`, race, vet, SQLC, PostgreSQL/Redis, OpenAPI, route inventory, Docker and HTTP smoke. Service tests cover token exchange failure classes, success, state single-use, TTL, PKCE, and malformed state. Existing Gin callback behavior tests must remain green. Keep as draft until CI results and implementation acceptance.
