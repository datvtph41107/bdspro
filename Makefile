DB_URL ?= postgres://root:secret@127.0.0.1:15432/bdspro?sslmode=disable

.PHONY: up infra ps logs down server dbstatus migrateup migrateversion new_migration sqlc mock fmt test test-integration

# Full application: Gin HTTP (host :18080), PostgreSQL (:15432), Redis (:16379).
up:
	docker compose up -d --build

# Infrastructure only, for running "make server" on the WSL host.
infra:
	docker compose up -d postgres redis

ps:
	docker compose ps

logs:
	docker compose logs --tail=100 -f

# Preserves PostgreSQL named volume. Do not use "down --volumes" on important data.
down:
	docker compose down

server:
	go run .

# Read-only inspection of the schema migration state.
dbstatus:
	docker compose exec -T postgres psql -U root -d bdspro -c "SELECT version, dirty FROM schema_migrations;"

# Requires locally installed migrate CLI; application startup also applies pending migrations.
migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateversion:
	migrate -path db/migration -database "$(DB_URL)" version

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/datvtph41107/bdspro/db/sqlc Store
	mockgen -package mockwk -destination worker/mock/distributor.go github.com/datvtph41107/bdspro/worker TaskDistributor

fmt:
	gofmt -w $$(git ls-files '*.go')

# Fast feedback; PostgreSQL write tests skip unless CI or an isolated test instance is authorized.
test:
	go test ./...

# Full suite against a disposable PostgreSQL container, never the development database.
test-integration:
	sh scripts/test-integration.sh
