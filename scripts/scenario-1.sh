#!/usr/bin/env bash
# Scenario 1 trigger of record: employee uploads customer_database.csv to ChatGPT.
# Reads the demo CSV and POSTs it to prompt-svc /prompts/inspect, then prints the result.
set -euo pipefail

PROMPT_SVC="${PROMPT_SVC:-http://localhost:8082}"
CSV="$(dirname "$0")/../fixtures/demo-data/customer_database.csv"

content="$(cat "$CSV")"

# Build the request body with python to JSON-escape the CSV safely.
body="$(python - "$content" <<'PY'
import json, sys
print(json.dumps({
    "userEmail": "james.wilson@example.com",
    "provider": "chatgpt",
    "content": sys.argv[1],
    "artifactName": "customer_database.csv",
    "contentType": "csv",
    "scenario": "scenario-1",
}))
PY
)"

echo ">> POST ${PROMPT_SVC}/prompts/inspect (customer_database.csv -> ChatGPT)"
curl -sS -X POST "${PROMPT_SVC}/prompts/inspect" \
  -H 'Content-Type: application/json' \
  --data-binary "$body" \
  --max-time 120 | python -m json.tool
