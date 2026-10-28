DB_URL ?= postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable
DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres

.PHONY: postgres redis createdb migrateup migrateup1 migratedown migratedown1 new_migration sqlc test server fmt

postgres:
	docker start $(POSTGRES_CONTAINER)

redis:
	docker start redis

createdb:
	docker exec $(POSTGRES_CONTAINER) createdb --username=root --owner=root $(DB_NAME)

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

test:
	go test -v ./...

server:
	go run .

fmt:
	gofmt -w .

.PHONY: swagger verify-api-contract
swagger:
	python3 doc/sync_swagger.py

verify-api-contract:
	go test ./internal/api -run 'TestOpenAPICoversAllRegisteredRoutes|TestSwaggerServedInDevelopmentOnly' -count=1
