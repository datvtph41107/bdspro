DB_URL=postgresql://root:secret@localhost:5432/bdspro?sslmode=disable
DIR_SQL=migrations

postgres:
	docker run --name postgres -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -p 5432:5432 -d postgres:18-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root bdspro

dropdb:
	docker exec -it postgres dropdb bdspro

sqlc:
	sqlc generate

# make newsql name=add_something
newsql:
	@test -n "$(name)" || (echo "Usage: make newsql name=add_something"; exit 1)
	migrate create -ext sql -dir $(DIR_SQL) -seq $(name)

# migrate force schema_migrations dirty
migratef:
	@test -n "$(version)" || (echo "Usage: make migrationforce version=N"; exit 1)
	migrate -path $(DIR_SQL) -database "$(DB_URL)" force $(version)

migrateup:
	migrate -path $(DIR_SQL) -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path $(DIR_SQL) -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path $(DIR_SQL) -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path $(DIR_SQL) -database "$(DB_URL)" -verbose down 1

.PHONY: postgres createdb dropdb sqlc newsql migrateup migratedown migrateup1 migratedown1