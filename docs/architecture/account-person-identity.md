# Account and Person identity in HTTP responses

An authenticated `Account` references one `Person` via
`accounts.person_id`. PostgreSQL is the canonical authority for that
relation; a JWT supplies only the account principal.

`services/convert.ToUser` now includes both `account_id` and the persisted
`person_id`. The same `modules/structs.UserResponse` is reused by
registration, login (password/OIDC/phone), and `GET /users/me`; no per-handler
handwritten identity map is maintained.

The `person_id` is a read-only relationship ID, **not** an authorization
credential. All account-scoped mutations must derive the acting account
from verified authentication middleware and enforce membership in models/SQL
rather than trusting an account/person ID supplied by clients.

This additive response field does not fabricate a profile name or establish
person identity verification. Editable `person_profiles`, `account_settings`
and other data require separate canonical schema/SQLC contracts.
