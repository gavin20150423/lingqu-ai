#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'docker runtime resources test failed: %s\n' "$1" >&2
  exit 1
}

assert_line() {
  file=$1
  line=$2
  grep -Fqx "$line" "$file" || fail "$file is missing: $line"
}

assert_count() {
  file=$1
  line=$2
  expected=$3
  actual=$(grep -Fxc "$line" "$file" || true)
  [ "$actual" -eq "$expected" ] || fail "$file has $actual occurrences of '$line', expected $expected"
}

test -s backend/resources/model-pricing/model_prices_and_context_window.json || \
  fail 'fallback pricing data is missing or empty'

assert_line Dockerfile.goreleaser 'COPY --chown=sub2api:sub2api backend/resources /app/resources'
assert_line deploy/Dockerfile 'COPY --from=backend-builder --chown=sub2api:sub2api /app/backend/resources /app/resources'
assert_count .goreleaser.yaml '      - backend/resources' 4
assert_count .goreleaser.simple.yaml '      - backend/resources' 1

for compose_file in \
  deploy/docker-compose.yml \
  deploy/docker-compose.local.yml \
  deploy/docker-compose.standalone.yml
do
  assert_line "$compose_file" '    cpus: "${SUB2API_CPU_LIMIT:-2.0}"'
  assert_line "$compose_file" '    mem_limit: "${SUB2API_MEMORY_LIMIT:-2g}"'
  assert_line "$compose_file" '    pids_limit: ${SUB2API_PIDS_LIMIT:-512}'
done

for compose_file in \
  deploy/docker-compose.yml \
  deploy/docker-compose.local.yml \
  deploy/docker-compose.standalone.yml \
  deploy/docker-compose.dev.yml
do
  assert_line "$compose_file" '      - VIDEO_API_ENABLED=${VIDEO_API_ENABLED:-false}'
  assert_line "$compose_file" '      - VIDEO_API_PUBLIC_BASE_URL=${VIDEO_API_PUBLIC_BASE_URL:-}'
  assert_line "$compose_file" '      - VIDEO_API_REQUEST_TIMEOUT_SECONDS=${VIDEO_API_REQUEST_TIMEOUT_SECONDS:-30}'
  assert_line "$compose_file" '      - VIDEO_API_RECONCILE_INTERVAL_SECONDS=${VIDEO_API_RECONCILE_INTERVAL_SECONDS:-30}'
  assert_line "$compose_file" '      - VIDEO_API_RECONCILE_BATCH_SIZE=${VIDEO_API_RECONCILE_BATCH_SIZE:-20}'
done

printf 'docker runtime resources test passed\n'
