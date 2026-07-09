#!/usr/bin/env bash
# Scenario 2 trigger: developer uploads source_code_leak.py to Claude.
# Expected: CODE (+ SECRET) categories; decision WARN or BLOCK; risk > benign baseline.
# Reads the Python source file and POSTs it to prompt-svc /prompts/inspect.
set -euo pipefail

PROMPT_SVC="${PROMPT_SVC:-http://localhost:8082}"
PY_FILE="$(dirname "$0")/../fixtures/demo-data/source_code_leak.py"

content="$(cat "$PY_FILE")"

# Build the request body with python to JSON-escape the file content safely.
body="$(python - "$content" <<'PY'
import json, sys
print(json.dumps({
    "userEmail": "dev.engineer@example.com",
    "provider": "claude",
    "content": sys.argv[1],
    "artifactName": "source_code_leak.py",
    "contentType": "code",
    "scenario": "scenario-2",
}))
PY
)"

echo ">> POST ${PROMPT_SVC}/prompts/inspect (source_code_leak.py -> Claude)"
curl -sS -X POST "${PROMPT_SVC}/prompts/inspect" \
  -H 'Content-Type: application/json' \
  --data-binary "$body" \
  --max-time 120 | python -m json.tool
