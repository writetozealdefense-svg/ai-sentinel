#!/usr/bin/env bash
# Reset the demo between talks: wipe datastore volumes, restart, re-seed.
# Usage: scripts/demo-reset.sh [--no-seed]   (~2 min; images are NOT rebuilt)
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE="$ROOT/deploy/compose"
SEED=1
[ "${1:-}" = "--no-seed" ] && SEED=0

cd "$COMPOSE"
echo "==> Stopping stack and dropping volumes (full data reset)"
docker compose down -v

echo "==> Starting fresh"
docker compose up -d

echo "==> Waiting for datastores + services"
sleep 5
for i in $(seq 1 60); do
  starting=$(docker compose ps --format '{{.Health}}' | grep -c 'starting' || true)
  [ "${starting:-0}" = 0 ] && break
  sleep 3
done
docker compose ps --format '  {{.Name}}\t{{.Status}}'

if [ "$SEED" = 1 ]; then
  echo "==> Re-seeding demo scenarios"
  bash "$ROOT/scripts/demo-seed.sh"
fi
echo "==> Reset complete. Dashboards at http://localhost:3100"
