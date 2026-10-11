# CORE-05: integrate with user-local BDSPro without data loss

Remote feature branch: feature/core05-source-intake, based on GitHub dev.

The user reported local Auth/Organization implementation not present on remote. Never overwrite it or assume matching migration versions.

## 1. Inspect local state before touching it

    cd ~/projects/bdspro
    git status --short --branch
    git branch -vv
    git fetch origin --prune
    git rev-list --left-right --count dev...origin/dev

Read the output before any merge/pull; fetch only changes remote-tracking refs.

## 2. Inspect the separate module branch without changing the active local tree

    git worktree add --detach ../bdspro-core05 origin/feature/core05-source-intake

Target path must not already exist. This creates a separate checkout; it does not merge the new code into the local dev checkout.

## 3. Reconcile dependencies

Feature code expects real tables/columns:
- organizations(party_id,archived_at), accounts(id,person_id,closed_at)
- permissions(code), organization_memberships(organization_id,person_id,joined_at,ended_at)
- organization_role_assignments(membership_id,organization_id,role_id,assigned_at,ended_at)
- organization_role_permissions(role_id,permission_code)
- organization_membership_blocks(membership_id,blocked_at,ended_at)

Check the local physical schema and FK/unique keys, not just canonical DBML. The CI fixture in db/source/testdata/core_prerequisites.sql exists only for isolated testing.

## 4. Integrate the migration

The module relative migration is 000001; it is NOT the next version in an established main migration history. In the local active repo create the proper next version:

    migrate create -ext sql -dir migrations -seq create_source_intakes_and_price_reports

Copy the UP and DOWN **contents**, not the filename/version, from ../bdspro-core05/db/source/migrations into the newly numbered pair. Do not use migrate -path db/source/migrations against the existing application database; the separate path may share/conflict with schema_migrations.

Apply to a designated disposable test DB with canonical prerequisites first. Verify the exact DB name and schema_migrations version before migrate up.

## 5. Merge source/service/router/config carefully

Copy or cherry-pick db/source/query, source, and sqlc.source.yaml as appropriate, and update the sqlc schema list to the actual local migrations. Keep separate generated db/sourcesqlc package or consolidate codegen deliberately. Run sqlc generate; do not hand-edit generated Go.

Wire source.NewHandler(source.NewService(pool), resolver) using only a trusted, validated Session-to-Account resolver. Do NOT resolve AccountID from a user-supplied header or request payload. Mount four routes under /organizations/ with the existing HTTP mux; remove conflicts rather than overwriting existing handlers.

## 6. Full local acceptance gate

Verify migration up/down on disposable DB, sqlc generated compilation, Go tests, safe retry under concurrency, org scope and role-block checks, initial-price rollback, wrong-organization composite FK, GET history authorization, denied/revoked account, and HTTP cookie/session integration. Add an app-level concurrency test for revoke-vs-write before claiming strict offboarding semantics.

The local-only Auth/Organization source and PostgreSQL state cannot be certified by a GitHub remote feature branch. An isolated green GitHub CI run is not proof of integrated local/production correctness. Keep PR draft until local prerequisites are committed, integrated and reviewed.
