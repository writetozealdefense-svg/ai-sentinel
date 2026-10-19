#!/usr/bin/env bash
# Seed the running stack with every demo scenario so all dashboards are populated
# before a talk. Idempotent enough to re-run (it just adds more activity).
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PROMPT=${PROMPT_SVC:-http://127.0.0.1:8082}
DISCOVERY=${DISCOVERY_SVC:-http://127.0.0.1:8091}
AGENT=${AGENT_SVC:-http://127.0.0.1:8083}
COMPLIANCE=${COMPLIANCE_SVC:-http://127.0.0.1:8084}

wait_for() { # name url
  local name=$1 url=$2 i
  printf '  waiting for %-14s' "$name"
  for i in $(seq 1 60); do
    if curl -sf -o /dev/null "$url/healthz" 2>/dev/null; then echo " ready"; return 0; fi
    sleep 2
  done
  echo " TIMEOUT"; return 1
}

echo "==> Waiting for services"
wait_for prompt-svc     "$PROMPT"     || exit 1
wait_for discovery-svc  "$DISCOVERY"  || exit 1
wait_for agent-svc      "$AGENT"      || exit 1
wait_for compliance-svc "$COMPLIANCE" || exit 1

echo "==> Seeding prompt/discovery scenarios (real detection; first run warms Presidio)"
for s in 1 2 3 4 6 7; do
  [ -f "scripts/scenario-$s.sh" ] || { echo "  scenario-$s : (missing, skipped)"; continue; }
  out=$(bash "scripts/scenario-$s.sh" 2>/dev/null)
  dec=$(printf '%s' "$out" | grep -oE '"decision":[[:space:]]*"[A-Z]+"' | head -1 | grep -oE '[A-Z]+"$' | tr -d '"')
  risk=$(printf '%s' "$out" | grep -oE '"riskScore":[[:space:]]*[0-9]+' | head -1 | grep -oE '[0-9]+$')
  printf "  scenario-%-2s: decision=%-6s risk=%s\n" "$s" "${dec:-n/a}" "${risk:-n/a}"
done

echo "==> Seeding agent-security scenario 5 (risk climbs to threshold breach)"
if [ -f scripts/scenario-5.sh ]; then
  out=$(bash scripts/scenario-5.sh 2>/dev/null)
  score=$(printf '%s' "$out" | grep -oE '"riskScore":[[:space:]]*[0-9]+' | tail -1 | grep -oE '[0-9]+$')
  printf "  scenario-5 : final agent riskScore=%s (threshold 70)\n" "${score:-n/a}"
fi

echo "==> Seed complete."
