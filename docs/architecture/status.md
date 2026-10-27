# Implementation and acceptance status — 2026-10-27

**Source baseline:** `dev` after the Gin-native routing and service-boundary
refactors. Determine the precise HEAD and CI result at review time; do not
infer production acceptance from a source tree or green compile alone.

| Ownership | Current source | What exists |
|---|---|---|
| Startup and shutdown | `main.go` | One Go process with HTTP and Asynq |
| HTTP runtime | `internal/core/router/server.go` | Gin, ingress, CORS, timeouts |
| HTTP contract | `internal/routes/api.go` | One route file with conditional endpoints |
| Auth settings | `internal/config/auth.go` | One JWT Account principal; no unused guards |
| Handlers | `internal/api/handler/` | Binding, admission, HTTP outputs |
| Operations | `internal/api/service/` | Auth/session, Organization, Google token, Phone OTP |
| Error/response | `internal/core/errors`, `internal/core/response` | Safe known errors and one JSON writer |
| Database | `db/query`, `db/sqlc`, `db/migration` | SQLC, PostgreSQL transactions |
| Async | `internal/worker/`, `internal/jobs/`, `internal/mail/` | Email verification worker |
| CI | `.github/workflows/test.yml` | PostgreSQL migrations and `make test` |

## Verified scope and remaining limits

- The architecture does not introduce `/api/v1`. Existing URLs,
  OAuth redirect URI, rate-limit windows and middleware scopes remain
  compatibility constraints.
- AuthService owns email verification, session renewal and revocation;
  OrganizationService owns input business validation and database error
  classification. HTTP handlers remain responsible for parsing requests.
- Google OAuth callback includes browser redirects and cookies and does
  not fit a generic JSON-only response writer.
- No `admin` principal, guard, password/session backend or `internal/routes/admin.go`
  should be invented before real authorization requirements.
- Focused JWT middleware, session lifecycle and Organization authorization tests now exist under `internal/`. PostgreSQL authorization integration runs only with the dedicated disposable test database in CI. These tests do not replace provider E2E or production acceptance.
- Registration's PostgreSQL/Asynq handoff remains non-atomic.
  Phone-OTP replay, lockout and OAuth callback need explicit runtime
  verification in disposable environments before production acceptance.

## Gates

`gofmt`, `go vet ./...`, `go build ./...`, `go test ./...` and
behavioral HTTP/security tests are separate pieces of evidence. Do not
claim WSL or production smoke acceptance until executed.
