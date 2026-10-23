DB_URL ?= postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable
DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres

.PHONY: network postgres redis createdb migrateup migrateup1 migratedown migratedown1 new_migration db_docs db_schema sqlc mock proto fmt test test-integration server

network:
	docker network create bdspro-network

redis:
	docker run --name redis -p 6379:6379 -d redis:7-alpine

postgres:
	docker start $(POSTGRES_CONTAINER)

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

db_docs:
	dbdocs build doc/db.dbml

db_schema:
	dbml2sql --postgres -o doc/schema.sql doc/db.dbml

sqlc:
	sqlc generate

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/datvtph41107/bdspro/db/sqlc Store
	mockgen -package mockwk -destination worker/mock/distributor.go github.com/datvtph41107/bdspro/worker TaskDistributor

test:
	go test -v -cover ./...

test-integration:
	sh scripts/test-integration.sh

server:
	go run main.go

proto:
	protoc -I proto \
		--go_out=. --go_opt=module=github.com/datvtph41107/bdspro \
		--go-grpc_out=. --go-grpc_opt=module=github.com/datvtph41107/bdspro \
		--grpc-gateway_out=. --grpc-gateway_opt=module=github.com/datvtph41107/bdspro \
		--openapiv2_out=doc/swagger proto/bdspro.proto
	statik -src=./doc/swagger -dest=./doc

fmt:
	find . -type f -name '*.go' -not -path './.git/*' -print0 | xargs -0 gofmt -w
