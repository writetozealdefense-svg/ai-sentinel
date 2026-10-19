#!/usr/bin/env bash
# One command to a demo-ready stack: build (if needed) -> up -> wait healthy -> seed.
# Usage: scripts/demo-up.sh [--no-seed] [--no-build]
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE="$ROOT/deploy/compose"
SEED=1; BUILD=1
for a in "$@"; do
  case "$a" in
    --no-seed) SEED=0 ;;
    --no-build) BUILD=0 ;;
  esac
done

if ! docker ps >/dev/null 2>&1; then
  echo "!! Docker engine is not responding. Start Docker Desktop and re-run."
  echo "   (Recovery on this machine: kill all *docker* processes, relaunch Docker Desktop,"
  echo "    wait until 'docker ps' works — it can take a few minutes.)"
  exit 1
fi

cd "$COMPOSE"
[ -f .env ] || cp .env.example .env

if [ "$BUILD" = 1 ]; then
  echo "==> Building images (first run pulls the spaCy model; ~3-6 min)"
  docker compose build || { echo "!! build failed"; exit 1; }
fi

echo "==> Starting the stack"
docker compose up -d || { echo "!! compose up failed"; exit 1; }

echo "==> Waiting for containers to report healthy"
for i in $(seq 1 60); do
  bad=$(docker compose ps --format '{{.Name}} {{.Health}}' | grep -Ev 'healthy|^$' | grep -Evc 'starting' || true)
  # Some services have no healthcheck (Health empty) — count only explicit 'unhealthy'/'starting'
  starting=$(docker compose ps --format '{{.Health}}' | grep -c 'starting' || true)
  [ "${starting:-0}" = 0 ] && break
  sleep 3
done
docker compose ps --format '  {{.Name}}\t{{.Status}}'

if [ "$SEED" = 1 ]; then
  echo "==> Seeding demo scenarios"
  bash "$ROOT/scripts/demo-seed.sh"
fi

echo
echo "==> Demo stack is up. Dashboards at http://localhost:3100"
echo "    Run 'bash scripts/demo-status.sh' for a readiness check."
