#!/bin/sh
set -eu

script="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)/safe-blue-green-cutover.sh"
test -f "$script" || { echo "missing $script" >&2; exit 1; }
command -v bash >/dev/null 2>&1 || exit 0
bash -n "$script"
grep -Fq 'flock -n 9' "$script"
grep -Fq -- '--address 127.0.0.1:2019' "$script"
grep -Fq 'runtime_has_upstream "$NEW_CONTAINER"' "$script"
grep -Fq 'docker stop "$OLD_CONTAINER"' "$script"
grep -Fq 'wait_public_health "$OBSERVATION_ROUNDS"' "$script"
echo 'safe blue/green cutover guard checks passed'
