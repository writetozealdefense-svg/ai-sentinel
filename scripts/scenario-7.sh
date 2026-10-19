#!/usr/bin/env bash
# Scenario 7 trigger (the GCC moment): a Riyadh bank employee pastes
# saudi_customer_data.csv — Saudi national IDs, iqama numbers, Saudi mobiles,
# SA IBANs — into ChatGPT (foreign-hosted).
#
# Expected: PII findings (Presidio EMAIL/PHONE/IBAN + regex SAUDI_NATIONAL_ID /
# SAUDI_MOBILE); decision BLOCK; violations mapped to REAL Saudi control IDs
# from fixtures/compliance:
#   PDPL      Article 29  (cross-border transfer, critical)
#   NCA_ECC   2-7-2       (classification-based data protection, critical)
#   SAMA_CSF  3.3.8       (infrastructure security incl. DLP 6a, critical)
#   SDAIA_AI  GAIG-GOV-ProhibInput (GenAI prohibited inputs, critical)
set -euo pipefail

PROMPT_SVC="${PROMPT_SVC:-http://localhost:8082}"
COMPLIANCE_SVC="${COMPLIANCE_SVC:-http://localhost:8084}"
CSV="$(dirname "$0")/../fixtures/demo-data/saudi_customer_data.csv"

content="$(cat "$CSV")"

# Build the request body with python to JSON-escape the CSV safely.
body="$(python - "$content" <<'PY'
import json, sys
print(json.dumps({
    "userEmail": "fatima.alzahrani@example.com",
    "provider": "chatgpt",
    "content": sys.argv[1],
    "artifactName": "saudi_customer_data.csv",
    "contentType": "csv",
    "scenario": "scenario-7",
}))
PY
)"

echo ">> POST ${PROMPT_SVC}/prompts/inspect (saudi_customer_data.csv -> ChatGPT)"
curl -sS -X POST "${PROMPT_SVC}/prompts/inspect" \
  -H 'Content-Type: application/json' \
  --data-binary "$body" \
  --max-time 120 | python -m json.tool

echo ""
echo ">> GET ${COMPLIANCE_SVC}/compliance/violations?framework=PDPL (latest Saudi violations)"
curl -sS "${COMPLIANCE_SVC}/compliance/violations?framework=PDPL&limit=5" \
  --max-time 30 | python -m json.tool \
  || echo "(compliance-svc not host-mapped on ${COMPLIANCE_SVC} — view via the Compliance dashboard instead)"
