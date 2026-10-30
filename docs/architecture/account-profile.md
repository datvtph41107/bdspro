# Account and person profile

The authenticated principal is an Account, linked to exactly one Person. A Person can participate in Organizations but does not automatically receive organization permissions or property rights.

Migration 000012 adds person_profiles and account_settings. Existing Person and Account rows receive empty profile names and default preferences; these are not fabricated identity details. Password, Google and phone account creation transactions each create the two records atomically with the account. The migration does not change legacy credential/session tables.

The source of truth for password/phone/Google availability remains the credential tables. GET /users/me projects only booleans, never credential hashes or provider subject IDs. Profile and settings updates derive account_id from the verified JWT, not from client input. Profile names are not legal identity evidence.

API operations:
- GET /users/me
- PATCH /users/me/profile (partial update: omitted name fields remain unchanged; an explicit empty string clears a field; an empty patch is rejected)
- GET /users/me/settings
- PATCH /users/me/settings

The registered user and login resources also expose person_id. Other existing fields remain unchanged. Avatar/file storage, device registration, login history and legal verification are not part of this slice; no fake avatar URL is returned.

Acceptance includes SQLC reproducibility, legacy account backfill, existing password/phone/Google registration behavior, two-account HTTP isolation, settings validation and end-to-end PostgreSQL tests. The profile tables update updated_at during writes; event and credential tables have different timestamp requirements and are not blindly modified.
