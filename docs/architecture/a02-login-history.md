# A02/2 — Password login history

This PR is stacked on A01 (PR #81, migration 000012), has migration 000013, and targets `dev`. Merge A01 first; afterward the diff should contain only A02/2 changes. It does **not** replace A02/1 (trusted ingress) or A03/1 (bcrypt login timing).

## Source of truth and behavior

- `account_login_events` is append-only, with `occurred_at` rather than an artificial `updated_at`. Its FK allows `NULL` for an unknown account and uses `ON DELETE SET NULL` for account deletion.
- Successful and rejected password logins are recorded with observed client IP and truncated User-Agent, up to 64 and 512 Unicode characters respectively. **No attempted email, password, refresh token, OTP or provider subject is recorded.** Unknown email attempts have null account_id.
- Ordinary credential failures still map to `auth.invalid_credentials`; persistence errors cause an internal failure instead of returning an ordinary 401 or issuing credentials.
- A login that created a session but could not write its audit row attempts to revoke the session before returning an error. It is not an atomic dual-write, and this PR does **not** claim it is.
- `GET /users/me/activity` derives AccountID from the authenticated access token and reads only that account's events (up to 50 latest). It must not accept account_id from the request.
- The API reflects the HTTP ingress IP; it does not claim IP/User-Agent are identity proof, and it depends on A02/1 reverse-proxy configuration when behind a trusted ingress.
- Phone OTP and Google sign-in audit, device registration/trust and access JWT invalidation after session revocation are outside this sub-slice.

## Retention and privacy

Historical source IP and User-Agent are sensitive security metadata. Read requires bearer authentication. The `DeleteExpiredLoginEvents` SQLC query removes rows older than 90 days and is tested. **No automatic retention scheduler is installed in this PR**: operators must schedule it or implement the durable maintenance job before claiming production retention compliance. Log statements must never include submitted credentials.

## Acceptance

- Migration upgrades safely from A01 schema 000012 to 000013, and rollback drops only the new audit table.
- Generated SQLC source is checked for drift in GitHub Actions.
- Service tests cover known/unknown-account failures, successful logins, bounded metadata, account isolation, and fail-closed audit outage.
- HTTP/PostgreSQL test covers stored events, returned resource fields, unauthenticated access, and retention SQL.
- Full CI: formatting, SQLC generation, OpenAPI, Go tests/race/vet/build, and HTTP smoke.
- No merge without latest-head green CI, privacy review and a separate operational plan for retention.
