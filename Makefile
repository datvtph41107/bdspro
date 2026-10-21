# BDSPro Go + PostgreSQL. Targets follow the SimpleBank workflow.
# DB_URL must be passed explicitly; never default a migration to production.
DB_URL ?=
DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres

.PHONY: postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 sqlc proto test testsqlc testsqlc-race server newmigration

postgres:
	docker run --name $(POSTGRES_CONTAINER) -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:18-alpine

createdb:
	docker exec $(POSTGRES_CONTAINER) createdb --username=root --owner=root $(DB_NAME)

dropdb:
	@case "$(DB_NAME)" in *_ci) ;; *) echo 'dropdb requires a disposable *_ci database'; exit 1 ;; esac
	@test "$(CONFIRM_DROP)" = "yes" || (echo 'Set CONFIRM_DROP=yes to drop $(DB_NAME)'; exit 1)
	docker exec $(POSTGRES_CONTAINER) dropdb --username=root $(DB_NAME)

newmigration:
	@test -n "$(name)" || (echo 'Usage: make newmigration name=change_name'; exit 1)
	migrate create -ext sql -dir db/migration -seq $(name)

migrateup:
	@test -n "$(DB_URL)" || (echo 'Set DB_URL=postgresql://.../database?sslmode=disable'; exit 1)
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	@test -n "$(DB_URL)" || (echo 'Set DB_URL explicitly'; exit 1)
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

# Preserve the prior single-step safety behavior for developer databases.
migratedown:
	@test -n "$(DB_URL)" || (echo 'Set DB_URL explicitly'; exit 1)
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

migratedown1: migratedown

sqlc:
	sqlc generate

# Buf generates versioned messages, gRPC stubs and RPC-only Swagger v2.
# Tools: buf + pinned protoc-gen-go, protoc-gen-go-grpc, protoc-gen-openapiv2.
# Never edit generated pb files or docs/grpc-swagger outputs.
proto:
	buf lint
	buf build
	buf generate

test:
	go test -v -cover ./...

testsqlc:
	@test -n "$(SQLC_TEST_DATABASE_URL)" || (echo 'Set SQLC_TEST_DATABASE_URL'; exit 1)
	go test -v -cover ./db/sqlc -count=1

testsqlc-race:
	@test -n "$(SQLC_TEST_DATABASE_URL)" || (echo 'Set SQLC_TEST_DATABASE_URL'; exit 1)
	go test -race ./db/sqlc -count=1 -timeout=120s -v

server:
	go run .
