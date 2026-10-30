# CODE-01 — HTTP request validation reference

## Purpose

Validate a concrete HTTP boundary in BDSPro without replacing Gin, inventing
a validation framework or changing the Organization/Workspace domain model.

Scope: existing `POST /organizations`, `PATCH /organizations/:id`, and
`POST /organizations/:id/members`. Public paths and successful behavior stay
unchanged. Domain and PostgreSQL errors are deliberately out of scope.

## Responsibility

1. Gin `ShouldBindJSON` parses input and executes struct binding rules.
2. `response.Validation(c, err, &req)` renders binding failures using public
   JSON field tags. It does not serialize raw Go or driver errors.
3. Handler validates the operation-specific blank-name rule before calling a
   service. The service remains defensive for non-HTTP callers.
4. Service owns ownership and other business invariants.
5. SQLC / PostgreSQL remain unchanged.

The current unversioned API uses HTTP 400 for invalid request input; CODE-01
preserves that HTTP status. Existing JSON `error` and `code` keys remain.
Detailed invalid fields use the additive `errors` object, with outer code
`validation_failed`. Body syntax/type failures have stable distinct codes.

Example:

```json
{
  "error": "validation failed",
  "code": "validation_failed",
  "errors": {
    "name": [{"code": "required", "message": "name is required"}]
  }
}
```

## Deliberate boundaries and tradeoffs

- This PR is an HTTP reference migration, not a rewrite of other endpoints.
  Old endpoints still use `response.Invalid`; their public error contract is
  not silently changed. A later reviewed compatibility PR may migrate them.
- The json-tag resolver handles the flat Request structs used in this slice.
  Nested DTO paths need an explicit additional test and implementation before
  being supported; do not claim full recursive coverage.
- Business errors and PostgreSQL constraints are not classified by this
  renderer. Unknown binding errors are safely masked.
- No URL rename is performed; Company/Workspace/Organization semantics are
  not interchangeable without a domain decision.
- HTTP errors for invalid input are returned before `AccountID(c)` or
  service operations in these handlers. Authentication middleware still runs
  on authenticated routes before the handler; this test does not bypass it.

## Verification

```sh
go test ./internal/api/response ./internal/api/handler
go test ./...
go test -race ./...
go vet ./...
go build ./...
make swagger
```

Acceptance requires the PR CI result for its exact HEAD SHA; a code review or
a green baseline CI run is not acceptance evidence.
