# G1.2c.9b — Detect organization ownership corruption

Baseline: BDSPro `dev@c7ca7222e2fdbbd59123957989c0596ee38eb69f`
Reference: [Gitea cmd/doctor.go](https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/cmd/doctor.go)

## Why the check exists

Migration v31 prevents an active owner from outliving its Account via
`organization_members.owner_person_id` foreign key. It does **not**
guarantee at least one owner per organization when a privileged raw
UPDATE atomically sets `role='member', owner_person_id=NULL` for the
last owner. The service transaction checks owner count and serializes
legitimate API role changes, but a DBA or new unreviewed raw SQL writer
could still perform that update.

The read-only `bdspro doctor` command detects these incidents. It does
not prevent them or infer an authorized replacement owner. Treat an
ownerless organization as a security/data incident requiring review.

## How to run

```bash
make doctor
# or from a deployed binary with reviewed app.ini
bdspro --config /path/to/app.ini doctor
```

No HTTP server, Redis or Asynq dependency. A non-zero command exit
denotes database failure or detected ownerless organizations; operators
should capture the error and alert, not discard it. Run after migrations
and periodically through the existing operations scheduler.

SQLC `ListOrganizationsWithoutActiveOwner` is read-only, ORDER BY
party ID, LIMIT 21. Only the first 20 IDs are included in diagnostics.
When the 21st exists, the result says *at least 21*, not an invented
exact count. No pagination of potentially sensitive membership data and
no unbounded in-memory list. This is an O(organization-count) scan in
the worst case; do not run it on every HTTP request.

## Ownership and Gitea comparison

Gitea provides a set of registered doctor checks and optional fixes.
BDSPro has exactly one check at present, so introducing a registry,
new middleware or a service that just forwards one SQLC call would
add indirection without value. CLI stays in `cmd/`; the bounded,
typed query and read-only report stay in `models/organization/`.

## Contract and limitations

- Healthy: zero ownerless organizations, exit 0.
- Corrupted: one or more ownerless organizations, exit nonzero with
  bounded IDs. No automatic demotion/promotion or destructive repairs.
- PostgreSQL unavailable: exit nonzero, not a misleading healthy result.
- G1.2c.9a account-link FK remains enforced separately.
- No claim that this detects suspended accounts, missing credentials,
  lost devices or unreachable verified contact channels.
- No claim of DB-level cross-row owner-count enforcement or p95/p99
  improvement. Benchmark diagnostic scans separately when needed.

Next architectural decision: owner-count enforcement at DB boundary
would need a verified referential anchor/constraint design or privileges
that genuinely prevent out-of-band writes, not a CHECK containing a
cross-row query and not a silent self-appointed owner.
