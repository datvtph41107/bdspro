# BDSPro — SimpleBank Lecture 37 baseline

This is a deliberately isolated **learning baseline**, not a production BDSPro release.
Reference: [SimpleBank Lecture 37 commit 5663a678](https://github.com/techschool/simplebank/commit/5663a678fa15e8427619a997af5ed6dbd6f31e33).
The original BDSPro history and its broader modules remain available on the `dev` branch.

## Scope

- Gin `api.Server` owns exactly `config`, `store`, `tokenMaker`, and `router`.
- `db.Store` + `db.SQLStore` and generated `db/mock.MockStore`.
- Create user (Party/Person/Account transaction), login (password + access/refresh JWT + SQL session), renew access token, protected account identity endpoint.
- No Organization, Source, SMS OTP, SMTP verification, Swagger or gRPC.
- API errors use direct `ctx.JSON`; no shared error contracts.

## New database only

The migration in this branch is an **isolated replacement baseline**, not an incremental migration for databases created by the existing `dev` branch. **Do not migrate an existing BDSPro database using it.**

Prerequisites: Go compatible with go.mod, PostgreSQL, `migrate`, `sqlc`, `mockgen` for regeneration.

```bash
docker start postgres
docker exec postgres createdb -U root -O root bdspro_l37
migrate -path db/migration -database "postgres://root:secret@localhost:5432/bdspro_l37?sslmode=disable" up
go mod tidy
make server
```

Routes:

- `POST /users` with JSON `{"email":"a@example.test","password":"strong-password"}`
- `POST /users/login` with matching credentials returns access token, refresh token and session ID
- `POST /tokens/renew_access` with `{"refresh_token":"..."}`
- `GET /users/me` with `Authorization: Bearer <access-token>`

Only use test credentials. This local lab stores plaintext signed refresh tokens in Postgres to follow the original lecture; a deployed application must review token storage, revocation, throttling, transport security, and privacy.

## Tests

```bash
go test -v ./api ./token ./util
docker exec postgres createdb -U root -O root bdspro_lecture37_test
migrate -path db/migration -database "postgres://root:secret@localhost:5432/bdspro_lecture37_test?sslmode=disable" up
DB_TEST_URL="postgres://root:secret@localhost:5432/bdspro_lecture37_test?sslmode=disable" make testdb
```

A plain `go test ./...` skips PostgreSQL tests when `DB_TEST_URL` is absent. That is **not** an acceptance of DB tests. For a clean source-of-truth check run `make sqlc`, `make mock`, `go mod tidy`, then `go test ./...`; inspect generated diffs.

## Git sync

```bash
git fetch origin --prune
git switch --track origin/refactor/lecture-37-simplebank-baseline
git status --short --branch
```

Do not rewrite or force-push `dev`; do not apply these migrations to databases from the old branch.
