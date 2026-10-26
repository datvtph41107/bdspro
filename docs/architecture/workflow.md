# DEV-01 — Developer work formula and review ownership

## Start at the route

Open `routes/` and locate the endpoint by HTTP method/path. Confirm public/authenticated/protected scope and exact middleware order. Then open the named controller, input Body/Query/Path, service operation when needed, SQL query and output Resource. The *route* is the navigation index, not a backend policy engine.

## Repeatable feature sequence

1. Identify the current behavior and source SHA. Define actor, command, input, response and one allowed/denied case.
2. Put input structure and format/range validation in `app/http/requests/<area>`. Use `app/http/request.JSON(ctx,&body)` or `Query(ctx,&query)` explicitly in the Controller; never hide values in Gin context keys.
3. Declare the method/path in `routes/`; insert only the middleware needed for that endpoint; preserve source-rate-limit-before-bind where required.
4. Use a named controller method that receives validated input and verified actor. Keep it HTTP-specific.
5. For meaningful multi-step workflows or cross-transport reuse, implement an application service. For one atomic SQL statement, avoid ceremonial forwarding services.
6. Put all ordinary SQL in `db/query` and regenerate SQLC. Service owns operation order; database owns transactions, statement predicates and durable constraints.
7. Represent known business failures with canonical named errors; don't send raw errors. Return public Resource via one response writer.
8. If async work is required, service expresses scheduling intent; `worker` chooses transport policy and `jobs` executes.
9. Document changed behavior, environment, migrations, rollback, integration/CI limitations and code ownership; review git diff against the protected base.

## Source review gates

- **No added raw SQL in services/controllers**; no second `pgxpool.Pool` owner.
- **No error mapper switch per controller**; no exposure of backend error text to client.
- **No generic facade** unless it consolidates a demonstrated repeated mechanism.
- **No request struct inside controller** for new endpoints.
- **No hidden actor fallback**; missing Authenticate is a route contract failure.
- **No claim of complete acceptance** when integration environment/tests were skipped.
- **Tests:** current files are evidence, not automatic proof or design authority. Requalify rather than deleting coverage by name; a future test plan should target contracts/negative/concurrency cases and keep build/vet/smoke gates explicit.

## Evidence vs desire

Mark each architectural change as PROPOSED, IMPLEMENTED, CI-CHECKED or RUNTIME-ACCEPTED. A diagram or document does not establish runtime behavior. Use `docs/engineering/` for historical controls and `docs/architecture/` for the current target and migration ledger.
