# GitHub Actions and YAML ownership

The only active CI workflow is `.github/workflows/test.yml`.
It is an **executable acceptance pipeline**, not application configuration.
`docker-compose.yaml` describes the disposable development runtime;
`sqlc.yaml` describes the generator's SQL inputs/Go output.
Treat these as different schemas with different owners.

## How test.yml is interpreted

| YAML key | Actual operation | Reason |
| --- | --- | --- |
| `on.push`, `on.pull_request` | choose branches/PRs to test | never silently deploy or merge |
| `jobs.test.runs-on` | allocate Ubuntu runner | reproducible tool environment |
| `services.postgres`/`redis` | isolated dependencies with health gates | tests should not depend on someone's local stack |
| `steps.uses: checkout` | fetch PR commit | source under test is the PR head |
| `setup-go` | use `go.mod` Go version | avoid workstation-versus-CI drift |
| SQLC generation check | regenerate Go and reject diff | detect stale SQLC output |
| API contract check | compare OpenAPI methods to Gin routes | reject undocumented routes |
| migration tests | execute historical schema upgrade fixtures | reject known data loss regressions |
| Go checks | formatting, unit tests, race, vet, build | reproducible compile/static/exercised behavior |
| Docker build | build image, verify executable | catch missing files/instructions in image |
| smoke test | boot HTTP and call health/auth/documentation | verify basic request path |

The longer migration upgrade assertions previously inlined in the YAML
now live in **`tests/integration/migrations/`** as guarded shell scripts.
This buys an independent runnable test artifact and keeps the workflow
a readable *orchestration file*, not a second SQL source.
Both scripts require `BDS_TEST_MODE=disposable` and fixed disposable DB
names. They assume PostgreSQL and the migration CLI are available.

## Distinguish guarantees

A green workflow proves only the assertions the workflow actually runs.
It does not prove exactly-once jobs, safe production secrets, clean worker
shutdown, all API schemas, zero downtime or backup restorability.

CI must be tied to the commit SHA; a green earlier commit does not approve
the latest PR changes. No job should perform a git push or remote deploy.

Gitea references:
[Makefile](https://github.com/go-gitea/gitea/blob/main/Makefile),
[testing](https://github.com/go-gitea/gitea/blob/main/docs/testing.md),
[workflows](https://github.com/go-gitea/gitea/tree/main/.github/workflows).
