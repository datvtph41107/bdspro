DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres

.PHONY: config-local worker postgres redis createdb migrateup migrateup1 migratedown migratedown1 new_migration sqlc test server swagger

postgres:
	docker start $(POSTGRES_CONTAINER)

redis:
	docker start redis

createdb:
	docker exec $(POSTGRES_CONTAINER) createdb --username=root --owner=root $(DB_NAME)

migrateup: config-local
	go run . --config "$(CONFIG)" migrate

migrateup1: config-local
	go run . --config "$(CONFIG)" migrate --steps 1

migratedown: config-local
	go run . --config "$(CONFIG)" migrate --down

migratedown1: config-local
	go run . --config "$(CONFIG)" migrate --down --steps 1

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

test:
	go test -v ./...

CONFIG ?= custom/conf/app.ini

# Initialize a local developer config once. Never overwrite personal settings.
config-local:
	@test -f "$(CONFIG)" || { umask 077; mkdir -p "$(dir $(CONFIG))"; cp custom/conf/app.example.ini "$(CONFIG)"; }
	@echo "Config: $(CONFIG)"

server: config-local
	go run . --config "$(CONFIG)" web

worker: config-local
	go run . --config "$(CONFIG)" worker

swagger:
	swag init -g main.go --parseInternal -o doc/swagger --outputTypes json
	statik -src=./doc/swagger -dest=./doc
# Same application binary as Docker, but written to a local, ignored build directory.
.PHONY: build watch race vet
build:
	@mkdir -p bin
	go build -o bin/bdspro .

watch: config-local
	go run github.com/air-verse/air@v1.67.4 -c .air.toml

race:
	go test -race ./...

vet:
	go vet ./...
