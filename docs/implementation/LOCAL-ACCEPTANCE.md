# BDSPro CORE-01 → CORE-05: local end-to-end acceptance

**Safe synchronization: do not overwrite a local tree with unpublished work.** On your computer:

    cd ~/projects
    git clone --branch dev --single-branch https://github.com/datvtph41107/bdspro.git bdspro-core-complete
    cd bdspro-core-complete
    git status --short --branch
    git log -1 --oneline

The fresh checkout is separate from the existing ~/projects/bdspro working tree. Keep the older tree until you have compared and decided what to port; do not use reset --hard or force push as a shortcut.

## A. Isolated PostgreSQL database and reproducible build

The commands below are for local PostgreSQL 18; inspect before creating a database:

    docker exec postgres psql -U root -d postgres \
      -c "SELECT datname FROM pg_database WHERE datname='bdspro_core_dev';"

If no row, create one (once):

    docker exec postgres createdb -U root bdspro_core_dev

With golang-migrate and sqlc installed:

    export DB_URL='postgresql://root:secret@localhost:5432/bdspro_core_dev?sslmode=disable'
    migrate -path migrations -database "$DB_URL" -verbose up
    migrate -path migrations -database "$DB_URL" version
    sqlc generate -f sqlc.source.yaml
    go test ./...

Do not run migrations against old bdspro with Listing migration history. Docker credentials here are LAB defaults, not production secrets.

## B. Start the application in safe development mode

    export DATABASE_URL="$DB_URL"
    export APP_ENV=development
    export BIND_HOST=127.0.0.1
    export PORT=8080
    go run .

Another terminal, replace sample emails with dedicated .test lab addresses (not someone else's real inbox). Register three Accounts: An, Lan and reviewer:

    curl -sS -X POST http://127.0.0.1:8080/auth/register \
      -H 'Content-Type: application/json' \
      -d '{"email":"an@example.test","password":"LabLongPassword1234!"}'

In development on loopback only, response contains verification_token. Verify each registered Account with its own token:

    curl -i -X POST http://127.0.0.1:8080/auth/verify-email \
      -H 'Content-Type: application/json' \
      -d '{"token":"TOKEN_FROM_PREVIOUS_RESPONSE"}'

Repeat registration/verification for reviewer@example.test and lan@example.test.

Log in and retain cookies (the temporary cookie files contain session secrets; limit access and delete when finished):

    umask 077
    curl -i -c /tmp/bdspro-an.cookies -X POST http://127.0.0.1:8080/auth/login \
      -H 'Content-Type: application/json' \
      -d '{"email":"an@example.test","password":"LabLongPassword1234!"}'

Log in reviewer/Lan separately with -c /tmp/bdspro-reviewer.cookies and /tmp/bdspro-lan.cookies. Verify one cookie:

    curl -b /tmp/bdspro-an.cookies http://127.0.0.1:8080/auth/me

## C. Bootstrap reviewer in a disposable LAB only

A tenant cannot appoint a platform reviewer. The following command deliberately uses database-operator privilege and must NEVER be exposed as an API or adapted to arbitrary user input:

    docker exec postgres psql -U root -d bdspro_core_dev -v ON_ERROR_STOP=1 \
      -c "INSERT INTO platform_organization_reviewers(account_id,authorization_reference)
          SELECT a.id,'LAB-ONLY-operator-approval'
          FROM accounts a
          JOIN account_emails ae ON ae.account_id=a.id
          WHERE ae.email='reviewer@example.test' AND ae.verified_at IS NOT NULL;"

This verifies **only** that the lab Account has an email already marked verified; it does not prove that person is a real platform operator. Real reviewer onboarding must use authenticated operator administration and documented evidence.

## D. Register ABC, approve, invite Lan

An submits:

    curl -b /tmp/bdspro-an.cookies -H 'Content-Type: application/json' \
      -X POST http://127.0.0.1:8080/organizations/registrations \
      -d '{"name":"ABC Realty Lab","jurisdiction":"VN","registration_code":"LAB-ABC-2026"}'

Record returned request_id. Before reviewer approval, querying organizations in DB should not show an ABC organization.

Reviewer approves after inspecting lab evidence:

    curl -b /tmp/bdspro-reviewer.cookies -H 'Content-Type: application/json' \
      -X POST http://127.0.0.1:8080/platform/organizations/registrations/REQUEST_ID/approve \
      -d '{"review_reference":"LAB-ONLY-reviewed-company-identity"}'

Record returned organization_id. The operation atomically creates Party/Organization, RegistryIdentifier, Membership, scoped administrator Role and primary workspace administrator.

Find the role ID via lab SQL:

    docker exec postgres psql -U root -d bdspro_core_dev \
      -c "SELECT organization_id,id,name FROM organization_roles ORDER BY id;"

An invites Lan; use the organization and Role ID returned in the preceding steps:

    curl -b /tmp/bdspro-an.cookies -H 'Content-Type: application/json' \
      -X POST http://127.0.0.1:8080/organizations/ORG_ID/invitations \
      -d '{"email":"lan@example.test","role_id":ROLE_ID}'

Record invitation_id. Lan accepts:

    curl -b /tmp/bdspro-lan.cookies -X POST \
      http://127.0.0.1:8080/organizations/invitations/INVITATION_ID/accept

An or Lan can now call GET /organizations/ORG_ID/members. Being invited alone never granted membership before Accept.

## E. Capture unverified source and price history

Lan creates source and initial unverified asking price:

    curl -b /tmp/bdspro-lan.cookies -H 'Content-Type: application/json' \
      -H 'Idempotency-Key: lab-capture-001' \
      -X POST http://127.0.0.1:8080/organizations/ORG_ID/sources \
      -d '{"channel":"zalo","label":"reported house","area_m2":"75","initial_price":{"amount":"4200000000","currency":"VND","price_basis":"asking"}}'

Record source id. Repeat exact request with same Idempotency-Key -> 200 replay, not a duplicate row. Reusing that key with changed payload -> 409.

Append later price report:

    curl -b /tmp/bdspro-lan.cookies -H 'Content-Type: application/json' \
      -H 'Idempotency-Key: lab-price-002' \
      -X POST http://127.0.0.1:8080/organizations/ORG_ID/sources/SOURCE_ID/price-reports \
      -d '{"amount":"4100000000","currency":"VND","source_basis":"reconfirmed_call"}'

Read history:

    curl -b /tmp/bdspro-lan.cookies \
      http://127.0.0.1:8080/organizations/ORG_ID/sources/SOURCE_ID/price-reports

Expect both 4.2bn and 4.1bn statements; no new Property or Listing appears. The latest database insertion is not necessarily a source-authoritative current price.

## F. Negative evidence and cleanup

- Use Lan's cookie with a different Organization URL: no cross-org read/write.
- End a Membership or add an active MembershipBlock in the lab; the account loses its scoped data access.
- Invalid registration/duplicate email never leaves an orphan Party.
- Source capture with optional price is atomic; a missing price-report permission must roll back the source.
- Logout, then reuse the old Cookie: /auth/me returns 401.
- Preserve migration version, git HEAD, actual errors and query output in the acceptance notebook; do not declare production-ready from smoke tests.

Temporary local cookies contain bearer Session tokens; securely delete your lab cookie files when finished.

## G. What CI proves and what it cannot

.github/workflows/bdspro-core.yml executes migrations against disposable PostgreSQL, exercises end-to-end Go tests and checks generated sqlc/mod files for drift. It is engineering proof only; real legal identity evidence, email delivery retry, account recovery, admin dispute, external integration, production CSRF/rate limiting and local historical DB migration reconciliation remain separate gates.
