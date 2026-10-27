# Implementation and acceptance status — 2026-10-27

**Source baseline:** `dev` after the Gin-native routing and service-boundary
refactors. Determine the precise HEAD and CI result at review time; do not
infer production acceptance from a source tree or green compile alone.

| Ownership | Current source | What exists |
|---|---|---|
| Startup and shutdown | `main.go` | One Go process with HTTP and Asynq |
| HTTP runtime | `core/router/server.go` | Gin, ingress, CORS, timeouts |
| HTTP contract | `routes/api.go` | One route file with conditional endpoints |
| Auth settings | `config/auth.go` | One JWT Account principal; no unused guards |
| Handlers | `api/handler/` | Binding, admission, HTTP outputs |
| Operations | `api/service/` | Auth/session, Organization, Google token, Phone OTP |
| Error/response | `core/errors`, `core/response` | Safe known errors and one JSON writer |
| Database | `db/query`, `db/sqlc`, `db/migration` | SQLC, PostgreSQL transactions |
| Async | `worker/`, `jobs/`, `mail/` | Email verification worker |
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
- No `admin` principal, guard, password/session backend or `routes/admin.go`
  should be invented before real authorization requirements.
- There are currently no `*_test.go` source files in this checkout;
  `go test ./...` can prove compilation without providing behavioral
  coverage. A green CI run is not proof of auth, OTP, concurrency or
  end-to-end contract correctness.
- Registration's PostgreSQL/Asynq handoff remains non-atomic.
  Phone-OTP replay, lockout and OAuth callback need explicit runtime
  verification in disposable environments before production acceptance.

## Gates

`gofmt`, `go vet ./...`, `go build ./...`, `go test ./...` and
behavioral HTTP/security tests are separate pieces of evidence. Do not
claim WSL or production smoke acceptance until executed.
