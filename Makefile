DB_URL ?= postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable
DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres

.PHONY: config-local worker postgres redis createdb migrateup migrateup1 migratedown migratedown1 new_migration sqlc test server swagger

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

CONFIG ?= custom/conf/app.ini

# Initialize a local developer config once. Never overwrite personal settings.
config-local:
	@test -f "$(CONFIG)" || { mkdir -p "$(dir $(CONFIG))"; cp custom/conf/app.example.ini "$(CONFIG)"; }
	@echo "Config: $(CONFIG)"

server: config-local
	go run . --config "$(CONFIG)" web

worker: config-local
	go run . --config "$(CONFIG)" worker

swagger:
	swag init -g main.go --parseInternal -o doc/swagger --outputTypes json
	statik -src=./doc/swagger -dest=./doc