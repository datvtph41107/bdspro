#!/bin/sh
# Full Go test suite using a disposable PostgreSQL server.
# Does not attach to the application's Compose PostgreSQL volume.
set -eu

container="bdspro-integration-postgres"
test_db="postgres://root:secret@127.0.0.1:15433/bdspro?sslmode=disable"

if docker container inspect "$container" >/dev/null 2>&1; then
  echo "Refusing to reuse existing container: $container" >&2
  exit 1
fi

docker run --rm -d \
  --name "$container" \
  -p 127.0.0.1:15433:5432 \
  -e POSTGRES_USER=root \
  -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=bdspro \
  postgres:14-alpine -c cluster_name=bdspro-disposable >/dev/null

cleanup() {
  docker stop "$container" >/dev/null 2>&1 || true
}
trap cleanup EXIT HUP INT TERM

attempt=0
until docker exec "$container" pg_isready -U root -d bdspro >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 45 ]; then
    echo "Disposable PostgreSQL did not become ready" >&2
    exit 1
  fi
  sleep 1
done

docker run --rm \
  --network "container:$container" \
  -v "$PWD/db/migration:/migrations:ro" \
  migrate/migrate:v4.18.3 \
  -path /migrations \
  -database "postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable" up

BDSPRO_INTEGRATION_DB=1 DB_SOURCE="$test_db" go test -count=1 ./...
