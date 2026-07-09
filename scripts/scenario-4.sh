#!/usr/bin/env bash
# Scenario 4 trigger: HR admin uploads payroll sheet to Copilot.
# Expected: PCI (+ PII) categories; decision BLOCK; financial data alert.
# Reads the payroll CSV and POSTs it to prompt-svc /prompts/inspect.
set -euo pipefail

PROMPT_SVC="${PROMPT_SVC:-http://localhost:8082}"
CSV="$(dirname "$0")/../fixtures/demo-data/payroll.csv"

content="$(cat "$CSV")"

# Build the request body with python to JSON-escape the CSV safely.
body="$(python - "$content" <<'PY'
import json, sys
print(json.dumps({
    "userEmail": "hr.admin@example.com",
    "provider": "copilot",
    "content": sys.argv[1],
    "artifactName": "payroll.csv",
    "contentType": "csv",
    "scenario": "scenario-4",
}))
PY
)"

echo ">> POST ${PROMPT_SVC}/prompts/inspect (payroll.csv -> Copilot)"
curl -sS -X POST "${PROMPT_SVC}/prompts/inspect" \
  -H 'Content-Type: application/json' \
  --data-binary "$body" \
  --max-time 120 | python -m json.tool
