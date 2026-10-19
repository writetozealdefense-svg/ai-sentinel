#!/usr/bin/env bash
# Scenario 8 trigger (the Singapore moment): a Singapore bank employee pastes
# sg_customer_data.csv — NRIC/FIN numbers, SG mobiles, account numbers — into
# ChatGPT (foreign-hosted).
#
# Expected: PII findings (regex SG_NRIC_FIN / SG_MOBILE + Presidio EMAIL/PERSON);
# decision BLOCK; violations mapped to REAL Singapore control IDs from
# fixtures/compliance (source-cited in SOURCES-SG.md, review-gated):
#   PDPA_SG      s 26   (transfer limitation)        / s 24 (protection)
#   MAS_TRM      11.1.5 (unauthorised internet services carrying confidential data)
#   PDPC_GENAI   accountability of deployers for prompts/outputs
#   CSA / IMDA   agentic + AI security controls where categories intersect
set -euo pipefail
PROMPT_SVC="${PROMPT_SVC:-http://127.0.0.1:8082}"
CSV="$(dirname "$0")/../fixtures/demo-data/sg_customer_data.csv"

content="$(cat "$CSV")"
body="$(python - "$content" <<'PY'
import json, sys
print(json.dumps({
    "userEmail": "weilin.tan@example.com",
    "provider": "chatgpt",
    "content": sys.argv[1],
    "artifactName": "sg_customer_data.csv",
    "contentType": "csv",
    "scenario": "scenario-8",
}))
PY
)"

echo ">> POST ${PROMPT_SVC}/prompts/inspect (sg_customer_data.csv -> ChatGPT)"
curl -sS -X POST "${PROMPT_SVC}/prompts/inspect" \
  -H 'Content-Type: application/json' \
  --data-binary "$body" \
  --max-time 120 | python -m json.tool
