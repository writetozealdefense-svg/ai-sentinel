#!/usr/bin/env bash
# Scenario 5 trigger (PHASE3-INTEGRATION.md Interface 3): SIMULATED telemetry for
# agent finops-crew-01 (crewai). Expected score trajectory 12 -> 24 -> 36 -> 51 -> 76;
# event 5 crosses the 70 threshold and emits exactly ONE violation (NIST AI RMF,
# category OTHER, control from fixtures).
set -euo pipefail

AGENT_SVC="${AGENT_SVC:-http://localhost:8083}"

AGENT_ID="finops-crew-01"
FRAMEWORK="crewai"
OWNER="ops.lead@example.com"
NAME="FinOps Automation Crew"

BASE_EPOCH="$(date -u +%s)"

# ts N -> RFC3339 timestamp N seconds after the script started (keeps event
# ordering deterministic even when curls land within the same second).
ts() {
  local when=$((BASE_EPOCH + $1))
  if date -u -d "@${when}" +%Y-%m-%dT%H:%M:%SZ >/dev/null 2>&1; then
    date -u -d "@${when}" +%Y-%m-%dT%H:%M:%SZ          # GNU date
  else
    date -u -r "${when}" +%Y-%m-%dT%H:%M:%SZ           # BSD date
  fi
}

send() {
  local body="$1"
  curl -sS -X POST "${AGENT_SVC}/agents/telemetry" \
    -H 'Content-Type: application/json' \
    --data-binary "$body" \
    --max-time 30
  echo
}

echo ">> Scenario 5: agent telemetry for ${AGENT_ID} (${FRAMEWORK}) -> ${AGENT_SVC}/agents/telemetry"

echo ">> 1/5 tool_call jira (PROJ-1042)"
send "{\"agentId\":\"${AGENT_ID}\",\"framework\":\"${FRAMEWORK}\",\"owner\":\"${OWNER}\",\"name\":\"${NAME}\",\"action\":\"tool_call\",\"tool\":\"jira\",\"target\":\"PROJ-1042\",\"simulated\":true,\"timestamp\":\"$(ts 0)\"}"

echo ">> 2/5 tool_call github (payments-service)"
send "{\"agentId\":\"${AGENT_ID}\",\"framework\":\"${FRAMEWORK}\",\"owner\":\"${OWNER}\",\"name\":\"${NAME}\",\"action\":\"tool_call\",\"tool\":\"github\",\"target\":\"payments-service\",\"simulated\":true,\"timestamp\":\"$(ts 1)\"}"

echo ">> 3/5 tool_call salesforce (accounts API)"
send "{\"agentId\":\"${AGENT_ID}\",\"framework\":\"${FRAMEWORK}\",\"owner\":\"${OWNER}\",\"name\":\"${NAME}\",\"action\":\"tool_call\",\"tool\":\"salesforce\",\"target\":\"accounts API\",\"simulated\":true,\"timestamp\":\"$(ts 2)\"}"

echo ">> 4/5 data_access salesforce.accounts (sensitive)"
send "{\"agentId\":\"${AGENT_ID}\",\"framework\":\"${FRAMEWORK}\",\"owner\":\"${OWNER}\",\"name\":\"${NAME}\",\"action\":\"data_access\",\"tool\":\"salesforce\",\"target\":\"salesforce.accounts\",\"sensitive\":true,\"simulated\":true,\"timestamp\":\"$(ts 3)\"}"

echo ">> 5/5 privilege_escalation (github org admin) — crosses threshold 70"
send "{\"agentId\":\"${AGENT_ID}\",\"framework\":\"${FRAMEWORK}\",\"owner\":\"${OWNER}\",\"name\":\"${NAME}\",\"action\":\"privilege_escalation\",\"target\":\"github org admin\",\"simulated\":true,\"timestamp\":\"$(ts 4)\"}"

echo ">> GET ${AGENT_SVC}/agents/${AGENT_ID}"
curl -sS "${AGENT_SVC}/agents/${AGENT_ID}" --max-time 30 | python -m json.tool
