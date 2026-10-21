# BDSPro local Go + PostgreSQL tasks (SimpleBank style).
# Migrating or deleting a DB requires an explicit target and arguments.
DB_URL ?=
DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres
DB_DOCS_DATABASE_URL ?=

.PHONY: postgres createdb dropdb newmigration migrateup migrateup1 migratedown migratedown1 sqlc proto db_docs db_schema test testsqlc testsqlc-race server

postgres:
	docker run --name $(POSTGRES_CONTAINER) -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:18-alpine

createdb:
	docker exec $(POSTGRES_CONTAINER) createdb --username=root --owner=root $(DB_NAME)

dropdb:
	$(if $(and $(filter %_ci,$(DB_NAME)),$(filter yes,$(CONFIRM_DROP))),docker exec $(POSTGRES_CONTAINER) dropdb --username=root $(DB_NAME),$(error Refusing dropdb: DB_NAME must end in _ci and CONFIRM_DROP=yes))

newmigration:
	$(if $(name),migrate create -ext sql -dir db/migration -seq $(name),$(error Pass name=change_name))

migrateup:
	$(if $(DB_URL),migrate -path db/migration -database "$(DB_URL)" -verbose up,$(error Explicit DB_URL required))

migrateup1:
	$(if $(DB_URL),migrate -path db/migration -database "$(DB_URL)" -verbose up 1,$(error Explicit DB_URL required))

migratedown:
	$(if $(DB_URL),migrate -path db/migration -database "$(DB_URL)" -verbose down 1,$(error Explicit DB_URL required))

migratedown1: migratedown

sqlc:
	sqlc generate

proto:
	buf lint
	buf build
	buf generate

# Generate DBML from migrated Postgres, never treat conceptual DBML as DDL.
db_docs:
	go run ./cmd/dbdocs -out doc -dbml db.dbml

db_schema:
	pg_dump --schema-only --no-owner --no-privileges --schema=public --dbname="$(DB_DOCS_DATABASE_URL)" > doc/schema.sql

test:
	go test -v -cover ./...

testsqlc:
	go test -v -cover ./db/sqlc -count=1

testsqlc-race:
	go test -race ./db/sqlc -count=1 -timeout=120s -v

server:
	go run .
