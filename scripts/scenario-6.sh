#!/usr/bin/env bash
# Scenario 6 trigger: analyst uses DeepSeek (shadow AI discovery).
# Step 1: POST a discovery event to discovery-svc (DeepSeek shows as SHADOW AI in inventory).
# Step 2: POST deepseek_prompt.txt to prompt-svc so it flows through the pipeline.
# Expected: discovery event recorded; DeepSeek appears as shadow-AI provider in dashboard.
set -euo pipefail

PROMPT_SVC="${PROMPT_SVC:-http://localhost:8082}"
DISCOVERY_SVC="${DISCOVERY_SVC:-http://localhost:8091}"
PROMPT_FILE="$(dirname "$0")/../fixtures/demo-data/deepseek_prompt.txt"

TIMESTAMP="$(python - <<'PY'
from datetime import datetime, timezone
print(datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"))
PY
)"

# ── Step 1: Discovery event ──────────────────────────────────────────────────
discovery_body="$(python - "$TIMESTAMP" <<'PY'
import json, sys
print(json.dumps({
    "provider": "deepseek",
    "userEmail": "analyst@example.com",
    "source": "proxy",
    "action": "prompt",
    "artifactName": "deepseek_prompt.txt",
    "simulated": True,
    "timestamp": sys.argv[1],
}))
PY
)"

echo ">> POST ${DISCOVERY_SVC}/discover/events (DeepSeek shadow-AI discovery)"
curl -sS -X POST "${DISCOVERY_SVC}/discover/events" \
  -H 'Content-Type: application/json' \
  --data-binary "$discovery_body" \
  --max-time 120 | python -m json.tool

echo ""

# ── Step 2: Flow prompt through inspect pipeline ─────────────────────────────
prompt_content="$(cat "$PROMPT_FILE")"

inspect_body="$(python - "$prompt_content" <<'PY'
import json, sys
print(json.dumps({
    "userEmail": "analyst@example.com",
    "provider": "deepseek",
    "content": sys.argv[1],
    "artifactName": "deepseek_prompt.txt",
    "contentType": "text",
    "scenario": "scenario-6",
}))
PY
)"

echo ">> POST ${PROMPT_SVC}/prompts/inspect (deepseek_prompt.txt -> DeepSeek)"
curl -sS -X POST "${PROMPT_SVC}/prompts/inspect" \
  -H 'Content-Type: application/json' \
  --data-binary "$inspect_body" \
  --max-time 120 | python -m json.tool
