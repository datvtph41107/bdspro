DB_URL ?=
DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres

.PHONY: postgres createdb migrateup migratedown sqlc mock test testdb server

postgres:
	docker start $(POSTGRES_CONTAINER)

createdb:
	docker exec $(POSTGRES_CONTAINER) createdb --username=root --owner=root $(DB_NAME)

migrateup:
	$(if $(DB_URL),migrate -path db/migration -database "$(DB_URL)" -verbose up,$(error DB_URL is required))

migratedown:
	$(if $(DB_URL),migrate -path db/migration -database "$(DB_URL)" -verbose down 1,$(error DB_URL is required))

sqlc:
	sqlc generate

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/datvtph41107/bdspro/db/sqlc Store

test:
	go test -v -cover ./...

testdb:
	$(if $(DB_TEST_URL),go test -v -count=1 ./db/sqlc,$(error DB_TEST_URL is required))

server:
	go run .
