# A03 — Password login credential-existence hardening

## Invariant

The public password login operation must not deliberately reveal whether an email is absent, an account has no password credential (e.g. provider-only account), or the submitted password is wrong.

Previously, the first two cases returned before performing a password hash comparison, while an existing password account ran bcrypt. This is a practical timing side channel.

## Change

- Use one fixed valid, **unlinked** bcrypt hash at the same cost as `util.HashPassword` in the absent-account and absent-password branches.
- Keep the external `InvalidCredentials` error and successful session issuance unchanged.
- No extra repository/service abstraction, new state, credential storage, provider changes or SQL is introduced.
- Tests verify the hash/cost and matching public failure identity in all three cases.

## Important limits

This reduces an obvious processing-cost difference; it is **not** a constant-time guarantee end-to-end (database reads, network behavior and rate limiting remain variable). Do not interpret timing measurements on a single machine as cryptographic proof.

CI must run `go test ./internal/service -run 'Test(UnknownPasswordUsesProductionBcryptCost|LoginDoesNotDiscloseMissingEmailOrPasswordCredential)'` plus the complete existing tests, race detector, vet and build on this PR head. A03 credential linking, recovery, replay protection, refresh policies and cross-channel acceptance remain separate work.
