# R22 — Durable Team Invitation delivery

The model's single invitation-creation path now creates the row and its
`task:send_team_invite` outbox intent in **one transaction**.

No plaintext proof or email is persisted in the outbox. The payload is only
`{"invite_id": "<uuid>"}`. The model derives an opaque token using
`modules/secret.DeriveToken(signingKey, "team-invite:v1", inviteID)` and stores
only SHA-256(token) in `team_invites`. The worker regenerates the same
one-time token when the job is delivered, rechecks its hash, confirms that the
invitation has not been accepted, expired or revoked, and sends through the
configured SMTP sender.

A delivery failure triggers Asynq retry. Duplicate deliveries can occur as
part of at-least-once processing, but they carry the same token; redemption
itself remains guarded by PostgreSQL and succeeds at most once.

**Key operation:** Keep the invitation signing key stable until all pending
proofs and queued jobs have expired. Rotating the shared signing key while
invites are pending will make existing proofs undeliverable. Introduce
key-version storage and rotation support before adopting frequent key
rotation.

A Team invitation is not an Organization membership invitation. The
recipient must already be an active Organization member with a verified email.
The SQL that creates an invitation enforces this; so does redemption. A job
carries no bearer auth and cannot grant permissions on its own.

Verification:
- `TestTeamInviteRecordsOutboxWithoutPlaintext` checks atomic intent and
  secrecy in PostgreSQL.
- `TestTeamInviteJobRegeneratesProofWithoutPersistingPlaintext` verifies
  reproducibility, HTML escaping and email delivery.
- Existing model tests cover invitation replay and simultaneous acceptance.

## Authenticated API

`POST /teams/{team_id}/invites` queues a 7-day invitation to a verified,
active Organization member and returns only its ID and expiration (HTTP 202).
`POST /teams/invites/accept` redeems the email-delivered proof under the
caller account (HTTP 201), and `DELETE /teams/{team_id}/invites/{invite_id}`
revokes it (HTTP 204). `GET /teams/{team_id}/members` uses bounded SQL
pagination with max page size 100.

The URI team ID must match the invitation's team on revocation. A non-owner
cannot issue or revoke; a recipient without matching verified email and active
organization membership cannot redeem. The public JSON contract **never
contains the invitation token**. Avoid writing request bodies or raw tokens
into audit logs. The API has no browser landing page in this backend-only
release: recipients are instructed to sign in and submit the token to the
authenticated REST endpoint.

Routes are added to a separate R23 manifest, preserving the R01/R20 snapshots.
