DB_URL ?= postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable
DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres

.PHONY: postgres createdb migrateup migratedown sqlc mock proto test server

postgres:
	docker start $(POSTGRES_CONTAINER)

createdb:
	docker exec $(POSTGRES_CONTAINER) createdb --username=root --owner=root $(DB_NAME)

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

sqlc:
	sqlc generate

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/datvtph41107/bdspro/db/sqlc Store

test:
	go test -v -cover ./...

server:
	go run main.go

proto:
	protoc -I proto \
		--go_out=. --go_opt=module=github.com/datvtph41107/bdspro \
		--go-grpc_out=. --go-grpc_opt=module=github.com/datvtph41107/bdspro \
		--grpc-gateway_out=. --grpc-gateway_opt=module=github.com/datvtph41107/bdspro \
		--openapiv2_out=doc/swagger proto/bdspro.proto
