DB_URL ?= postgres://root:secret@127.0.0.1:15432/bdspro?sslmode=disable

server:
	go run main.go

test:
	go test -v -cover ./...

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/datvtph41107/bdspro/db/sqlc Store
	mockgen -package mockwk -destination worker/mock/distributor.go github.com/datvtph41107/bdspro/worker TaskDistributor
