#!/usr/bin/env bash
# One-glance demo readiness: service health + seeded data counts + dashboard URLs.
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

declare -A SVC=(
  [detection-svc]=http://127.0.0.1:8000
  [discovery-svc]=http://127.0.0.1:8091
  [prompt-svc]=http://127.0.0.1:8082
  [compliance-svc]=http://127.0.0.1:8084
  [agent-svc]=http://127.0.0.1:8083
  [graph-svc]=http://127.0.0.1:8085
  [copilot-svc]=http://127.0.0.1:8086
  [telemetry-generator]=http://127.0.0.1:8087
)

echo "== Service health =="
for name in detection-svc discovery-svc prompt-svc compliance-svc agent-svc graph-svc copilot-svc telemetry-generator; do
  if curl -sf -o /dev/null "${SVC[$name]}/healthz" 2>/dev/null; then
    printf "  %-22s OK\n" "$name"
  else
    printf "  %-22s DOWN\n" "$name"
  fi
done

echo "== Seeded data (Postgres) =="
docker exec sentinel-postgres psql -U sentinel -d sentinel -tAc \
  "SELECT 'prompts='||count(*) FROM prompts UNION ALL
   SELECT 'violations='||count(*) FROM violations UNION ALL
   SELECT 'discovery_events='||count(*) FROM discovery_events UNION ALL
   SELECT 'agents='||count(*) FROM agents;" 2>/dev/null | sed 's/^/  /' || echo "  (postgres not reachable)"

echo "== Copilot mode =="
curl -sf http://127.0.0.1:8086/healthz 2>/dev/null | sed 's/^/  /'; echo

echo "== Dashboards =="
cat <<'EOF'
  Executive   http://127.0.0.1:3100/
  SOC         http://127.0.0.1:3100/soc
  Compliance  http://127.0.0.1:3100/compliance
  Agents      http://127.0.0.1:3100/agents
  Risk Graph  http://127.0.0.1:3100/graph
  Copilot     http://127.0.0.1:3100/copilot
EOF
