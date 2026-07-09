#!/usr/bin/env bash
# Scenario 3 trigger: nurse uploads patient records CSV to Gemini.
# Expected: PHI (+ PII) categories; decision BLOCK; HIPAA violation logged.
# Reads the patient CSV and POSTs it to prompt-svc /prompts/inspect.
set -euo pipefail

PROMPT_SVC="${PROMPT_SVC:-http://localhost:8082}"
CSV="$(dirname "$0")/../fixtures/demo-data/patient_records.csv"

content="$(cat "$CSV")"

# Build the request body with python to JSON-escape the CSV safely.
body="$(python - "$content" <<'PY'
import json, sys
print(json.dumps({
    "userEmail": "nurse.joy@example.com",
    "provider": "gemini",
    "content": sys.argv[1],
    "artifactName": "patient_records.csv",
    "contentType": "csv",
    "scenario": "scenario-3",
}))
PY
)"

echo ">> POST ${PROMPT_SVC}/prompts/inspect (patient_records.csv -> Gemini)"
curl -sS -X POST "${PROMPT_SVC}/prompts/inspect" \
  -H 'Content-Type: application/json' \
  --data-binary "$body" \
  --max-time 120 | python -m json.tool
