#!/usr/bin/env bash
# Safe blue/green cutover for a Docker + Caddy deployment.
# The old application remains serving until runtime and public checks pass.

set -Eeuo pipefail

usage() {
  cat >&2 <<'EOF'
Usage:
  safe-blue-green-cutover.sh --old-container OLD --new-container NEW \
    --caddy-container CADDY --caddyfile HOST_CADDYFILE \
    --api-url https://api.example.com/health \
    --cdn-url https://cdn.example.com/health
EOF
  exit 2
}

OLD_CONTAINER=''
NEW_CONTAINER=''
CADDY_CONTAINER=''
CADDYFILE=''
API_URL=''
CDN_URL=''
OBSERVATION_ROUNDS="${OBSERVATION_ROUNDS:-3}"
OBSERVATION_DELAY="${OBSERVATION_DELAY:-3}"
LOCK_FILE="${RELEASE_LOCK_FILE:-/opt/gavin2api/.safe-blue-green.lock}"

while (($#)); do
  case "$1" in
    --old-container) OLD_CONTAINER="${2:-}"; shift 2 ;;
    --new-container) NEW_CONTAINER="${2:-}"; shift 2 ;;
    --caddy-container) CADDY_CONTAINER="${2:-}"; shift 2 ;;
    --caddyfile) CADDYFILE="${2:-}"; shift 2 ;;
    --api-url) API_URL="${2:-}"; shift 2 ;;
    --cdn-url) CDN_URL="${2:-}"; shift 2 ;;
    --help|-h) usage ;;
    *) echo "unknown argument: $1" >&2; usage ;;
  esac
done

[[ -n "$OLD_CONTAINER" && -n "$NEW_CONTAINER" && -n "$CADDY_CONTAINER" && -n "$CADDYFILE" && -n "$API_URL" && -n "$CDN_URL" ]] || usage
[[ "$OLD_CONTAINER" != "$NEW_CONTAINER" ]] || { echo 'old and new containers must differ' >&2; exit 2; }
[[ -f "$CADDYFILE" ]] || { echo "Caddyfile not found: $CADDYFILE" >&2; exit 2; }

exec 9>"$LOCK_FILE"
flock -n 9 || { echo "another release is active: $LOCK_FILE" >&2; exit 1; }

log() { printf '[safe-cutover] %s\n' "$*"; }
fail() { echo "[safe-cutover] ERROR: $*" >&2; exit 1; }

container_status() { docker inspect "$1" --format '{{.State.Status}}' 2>/dev/null || true; }
container_health() { docker inspect "$1" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}no-healthcheck{{end}}' 2>/dev/null || true; }

require_running_healthy() {
  local container="$1" health
  [[ "$(container_status "$container")" == running ]] || fail "$container is not running"
  health="$(container_health "$container")"
  [[ "$health" == healthy || "$health" == no-healthcheck ]] || fail "$container health is $health"
}

runtime_config() {
  docker exec "$CADDY_CONTAINER" wget -qO- --timeout=5 http://127.0.0.1:2019/config/ 2>/dev/null || true
}

runtime_has_upstream() { runtime_config | grep -Fq "$1"; }

reload_caddy_file() {
  local file_in_container="$1"
  docker exec "$CADDY_CONTAINER" caddy validate --config "$file_in_container" --adapter caddyfile >/dev/null
  docker exec "$CADDY_CONTAINER" caddy reload --address 127.0.0.1:2019 --config "$file_in_container" --adapter caddyfile >/dev/null
}

public_health() {
  local api cdn
  api="$(curl -ksS --max-time 10 -o /dev/null -w '%{http_code}' "$API_URL")"
  cdn="$(curl -ksS --max-time 10 -o /dev/null -w '%{http_code}' "$CDN_URL")"
  log "public health: api=$api cdn=$cdn"
  [[ "$api" == 200 && "$cdn" == 200 ]]
}

wait_public_health() {
  local rounds="$1" i
  for ((i = 1; i <= rounds; i++)); do
    public_health || return 1
    ((i == rounds)) || sleep "$OBSERVATION_DELAY"
  done
}

CANDIDATE_CADDY="/tmp/safe-blue-green-${NEW_CONTAINER}.Caddyfile"
ROLLBACK_CADDY="/tmp/safe-blue-green-${OLD_CONTAINER}.Caddyfile"

rollback() {
  local reason="$1" old_health
  log "rollback: $reason"
  if [[ "$(container_status "$OLD_CONTAINER")" != running ]]; then
    docker start "$OLD_CONTAINER" >/dev/null
  fi
  old_health="$(container_health "$OLD_CONTAINER")"
  if [[ "$old_health" != healthy && "$old_health" != no-healthcheck ]]; then
    for _ in {1..30}; do
      old_health="$(container_health "$OLD_CONTAINER")"
      [[ "$old_health" == healthy ]] && break
      sleep 2
    done
  fi
  [[ "$old_health" == healthy || "$old_health" == no-healthcheck ]] || fail "rollback blocked: old health is $old_health"
  docker cp "$CADDYFILE" "$CADDY_CONTAINER:$ROLLBACK_CADDY"
  docker exec "$CADDY_CONTAINER" sed -i "s/${NEW_CONTAINER}/${OLD_CONTAINER}/g" "$ROLLBACK_CADDY"
  grep -Fq "$OLD_CONTAINER" <(docker exec "$CADDY_CONTAINER" cat "$ROLLBACK_CADDY") || fail 'rollback config does not reference old container'
  reload_caddy_file "$ROLLBACK_CADDY"
  runtime_has_upstream "$OLD_CONTAINER" || fail "Caddy runtime did not switch back to $OLD_CONTAINER"
  wait_public_health "$OBSERVATION_ROUNDS" || fail 'public health did not recover after rollback'
  docker stop "$NEW_CONTAINER" >/dev/null 2>&1 || true
  log 'rollback complete; old container remains serving traffic'
}

on_error() {
  local code=$?
  trap - ERR
  rollback "cutover failed (exit $code)" || exit 1
  exit "$code"
}
trap on_error ERR

require_running_healthy "$OLD_CONTAINER"
require_running_healthy "$NEW_CONTAINER"

log 'copying and validating candidate Caddyfile inside Caddy'
docker cp "$CADDYFILE" "$CADDY_CONTAINER:$CANDIDATE_CADDY"
docker exec "$CADDY_CONTAINER" caddy validate --config "$CANDIDATE_CADDY" --adapter caddyfile >/dev/null

runtime_has_upstream "$OLD_CONTAINER" || fail "Caddy runtime is not serving old upstream: $OLD_CONTAINER"
grep -Fq "$NEW_CONTAINER" "$CADDYFILE" || fail "candidate Caddyfile does not reference new container: $NEW_CONTAINER"

log 'reloading Caddy through 127.0.0.1:2019'
reload_caddy_file "$CANDIDATE_CADDY"
runtime_has_upstream "$NEW_CONTAINER" || fail "Caddy runtime did not switch to $NEW_CONTAINER"
runtime_has_upstream "$OLD_CONTAINER" && fail "Caddy runtime still contains old upstream: $OLD_CONTAINER"

log 'verifying public traffic before stopping old container'
wait_public_health "$OBSERVATION_ROUNDS"

log 'checks passed; stopping old application container only now'
docker stop "$OLD_CONTAINER" >/dev/null
wait_public_health "$OBSERVATION_ROUNDS"
log 'cutover complete; old container is stopped but retained for rollback'
