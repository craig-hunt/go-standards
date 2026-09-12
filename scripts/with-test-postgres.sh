#!/usr/bin/env bash
# Runs a command against a throwaway PostgreSQL container that exists only for
# this run. The container is created, used, and removed whatever the command's
# outcome, so no test ever touches a database holding real data.
set -euo pipefail

readonly image="${POSTGRES_IMAGE:?POSTGRES_IMAGE must name a Postgres image}"
readonly database="demo"
readonly user="demo"
readonly container_port="5432/tcp"
readonly ready_attempts=60
readonly container="go-standards-test-$$-${RANDOM}"
password="$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')"
readonly password

cleanup() {
  docker rm --force "${container}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker run --detach --name "${container}" \
  --env "POSTGRES_USER=${user}" \
  --env "POSTGRES_PASSWORD=${password}" \
  --env "POSTGRES_DB=${database}" \
  --publish "127.0.0.1::${container_port}" \
  "${image}" >/dev/null

# The image starts a socket-only server to initialize, then restarts on TCP.
# Probing over TCP waits for the real server rather than the init one.
for _ in $(seq "${ready_attempts}"); do
  if docker exec "${container}" pg_isready --host 127.0.0.1 --username "${user}" --dbname "${database}" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "${container}" pg_isready --host 127.0.0.1 --username "${user}" --dbname "${database}" >/dev/null

host_port="$(docker port "${container}" "${container_port}" | head -n 1 | awk -F: '{print $NF}')"
export TEST_DATABASE_URL="postgres://${user}:${password}@127.0.0.1:${host_port}/${database}?sslmode=disable"

"$@"
