# BDSPro canonical core (Go + PostgreSQL 18 + golang-migrate + sqlc)
# Explicitly set DB_URL. No silent migration against old bdspro Listing databases.
DB_URL ?=
.PHONY: sqlc test testsqlc testsqlc-race migrateup migratedown newmigration

sqlc:
	sqlc generate

test:
	go test ./... -count=1

newmigration:
	@test -n "$(name)" || (echo 'Usage: make newmigration name=change_name'; exit 1)
	migrate create -ext sql -dir migrations -seq $(name)

migrateup:
	@test -n "$(DB_URL)" || (echo 'Usage: make migrateup DB_URL=postgresql://.../bdspro_core_dev?sslmode=disable'; exit 1)
	migrate -path migrations -database "$(DB_URL)" -verbose up

migratedown:
	@test -n "$(DB_URL)" || (echo 'Usage: make migratedown DB_URL=... (destructive on managed data)'; exit 1)
	migrate -path migrations -database "$(DB_URL)" -verbose down 1

# Set SQLC_TEST_DATABASE_URL explicitly. COMMIT tests only run on disposable
# bdspro_sqlc_ci; regular bdspro tests use rollback or skip.
testsqlc:
	@test -n "$(SQLC_TEST_DATABASE_URL)" || (echo 'Set SQLC_TEST_DATABASE_URL to an explicitly selected test DB'; exit 1)
	go test ./db/sqlc -count=1 -v

testsqlc-race:
	@test -n "$(SQLC_TEST_DATABASE_URL)" || (echo 'Set SQLC_TEST_DATABASE_URL to an explicitly selected test DB'; exit 1)
	go test -race ./db/sqlc -count=1 -timeout=120s
