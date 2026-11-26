DB_NAME ?= bdspro
POSTGRES_CONTAINER ?= postgres

.PHONY: config-local postgres redis createdb migrateup migrateup1 migratedown migratedown1 new_migration sqlc test server swagger

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

.PHONY: generated sqlc-check
generated: sqlc

sqlc:
	sqlc compile
	@mkdir -p models/db/sqlc
	@find models/db/sqlc -maxdepth 1 -type f -name '*.go' -delete
	sqlc generate
	go run ./tools/dbalias

sqlc-check:
	@$(MAKE) --no-print-directory generated
	@git diff --exit-code -- models/db/sqlc models/db/sqlc_types.gen.go
	@test -z "$$(git ls-files --others --exclude-standard models/db/sqlc models/db/sqlc_types.gen.go)"
	@test -z "$$(find models/db -maxdepth 1 -name '*.sql.go' -print -quit)"

test:
	go test -v ./...

CONFIG ?= custom/conf/app.ini

config-local:
	@test -f "$(CONFIG)" || { umask 077; mkdir -p "$(dir $(CONFIG))"; cp custom/conf/app.example.ini "$(CONFIG)"; }
	@echo "Config: $(CONFIG)"

server: config-local
	go run . --config "$(CONFIG)" web

.PHONY: swagger swagger-check
swagger: swagger-check

swagger-check:
	python3 -m json.tool templates/swagger/openapi.json > /dev/null
	python3 -c 'import json; d=json.load(open("templates/swagger/openapi.json")); p=d["paths"]; assert "/api/v1/health" in p; assert all(k in p["/api/v1/orgs/{org}/teams"] for k in ("get","post")); assert all(k in p["/api/v1/orgs/{org}/teams/{team}"] for k in ("get","patch")); assert "sessionBearer" in d["components"]["securitySchemes"]'

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
