# Account Person Profile and Settings (R31)

This implementation uses the current SQLC/Gin/Gitea-inspired ownership on dev; it does **not** cherry-pick old PR #81 based on internal/ and migration 000012.

- `Person` remains the identity/actor root linked to a Party; `person_profiles` stores **editable, unverified presentation names**, not legal proof.
- `Account` remains the authenticated login identity. `account_settings` stores per-account locale/timezone/currency, not organization settings.
- Migration `000017` backfills **all existing Person rows** and Account rows, with empty names and conservative defaults. It does not fabricate legal names or account creation history.
- Password, verified-OIDC and phone-first registration create the profile and settings through the **same SQLC transaction** as the Party/Person/Account. Any failure rolls all rows back.
- The current OAuth2 source is `oauth2_credentials`; password and phone checks read credential tables. API exposes only credential-presence flags, never hashes, phone numbers, issuer subjects or tokens.
- `GET /users/me` returns the authenticated account with nested profile/security/settings. `PATCH /users/me/profile`, `GET/PATCH /users/me/settings` are authenticated and derive AccountID from the verified access token, never URL or body.
- PATCH is sparse. An omitted field is preserved; an explicit empty name clears it; an empty patch is rejected. PostgreSQL updates individual columns atomically, so unrelated concurrent field changes do not overwrite each other.
- Validation and CHECK constraints currently accept locales `vi/en`, timezones `Asia/Ho_Chi_Minh/UTC`, currencies `VND/USD`. Extend these deliberately after requirements and data migrations, not ad hoc.
- Rolling deploy warning: an old application binary writing accounts **after** migration 000017 but **before** deployment of the updated signup paths would omit profile/settings. Coordinate migration with a short account-write maintenance window or only switch write traffic after the new binary is ready. There is no hidden trigger or second lazy persistence path.
- `created_at` tracks when the profile/settings rows are created, not when the underlying Person first existed. `updated_at` changes on successful PATCH.

Acceptance gates: SQLC deterministic generation, migration 000016→000017 backfill+rollback, PostgreSQL two-account isolation and concurrent sparse fields, password/OAuth2/phone registration, authenticated Gin endpoint and safe errors, OpenAPI and route inventory, race/vet/Docker CI. Device registration, activity history, avatars, legal identity verification and multi-tenant policies are **not** claimed complete.
