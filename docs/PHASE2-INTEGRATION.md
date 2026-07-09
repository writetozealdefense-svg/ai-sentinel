# Phase 2 Integration Contract (Breadth on the proven pipeline)

Phase 1 is DONE and passing (see VERIFICATION.md Gate 1). The full stack is running via
`deploy/compose/docker-compose.yml`. Phase 2 adds breadth **through the same pipeline** — no new
bespoke detection/decision code paths. New scenarios are just data + triggers that hit the
existing `prompt-svc /prompts/inspect`. Plus two new dashboards and the SIMULATED runtime feed.

## Golden rule
Scenarios 2/3/4 go through the EXISTING `POST /prompts/inspect` (detection-svc → risk → OPA →
persist → violation). Do NOT add scenario-specific branches anywhere. The only "new logic" is
the telemetry-generator (runtime events) and two frontend pages.

## Running stack (host ports; container ports in parens)
detection-svc 8000 · discovery-svc 8091(8081) · prompt-svc 8082 · compliance-svc 8084 ·
opa 8181 · frontend 3100(3000) · postgres 5432 · nats 4222 · neo4j 7687 · redis 6379.
NEW in Phase 2: **telemetry-generator 8087**.

## The four scenarios (each must produce REAL detection + a risk/decision + a dashboard change)
| # | Story | Provider | Dataset (fixtures/demo-data/) | Expected categories | Expected decision |
|---|-------|----------|-------------------------------|---------------------|-------------------|
| 2 | Source code → Claude (code leakage) | claude | `source_code_leak.<ext>` (real source, may embed a secret) | CODE (+ SECRET) | WARN or BLOCK; risk clearly > a benign prompt |
| 3 | Patient records → AI → PHI | gemini | `patient_records.csv` | PHI (+ PII) | BLOCK; HIPAA violation appears |
| 4 | Payroll sheet → AI → PCI/financial | copilot | `payroll.csv` | PCI (+ PII) | BLOCK; alert |
| 6 | DeepSeek usage → Shadow-AI discovery | deepseek | (discovery event; small prompt optional) | — | discovery event → deepseek shows as SHADOW AI in inventory |

Datasets must be SYNTHETIC and must genuinely trigger their target category in detection-svc.
For Scenario 3, PHI detection must actually fire (MRN/ICD-10/clinical terms) — enhance
detection-svc's PHI recognizers if needed so it is REAL, not scripted. HIPAA fixture controls map
`PHI`, so a PHI finding will (via the existing compliance mapping) produce a **HIPAA** violation —
that is the "HIPAA mapping shown" requirement; do not special-case it.

Each scenario gets a trigger script `scripts/scenario-N.sh` (mirrors `scenario-1.sh`) and a
documented expected output.

## Backend additions
### prompt-svc — NEW `GET /prompts` (contract: prompts.yaml `listPrompts`)
`GET /prompts?limit=&decision=` → array of `PromptSummary` (newest first):
`{promptId, userEmail, provider, artifactName, decision, riskScore, categories[], scenario, timestamp}`.
Reads the `prompts` table (+ derive `categories` from `findings`). Powers SOC live monitoring.

### telemetry-generator — NEW service (Go, port 8087; contract: runtime.yaml)
- Emits SIMULATED Falco/Tetragon-style runtime-security events on an internal timer into an
  in-memory ring buffer (e.g. one event every ~2–4s; keep last ~200).
- `GET /runtime/events?limit=&source=` → newest-first array of `RuntimeEvent`
  `{eventId, source: falco|tetragon, rule, priority: info|notice|warning|error|critical, message,
   namespace, pod, container, simulated: true, timestamp}`.
- `GET /healthz` → `{"status":"ok","service":"telemetry-generator"}`.
- Realistic rules, e.g.: "Terminal shell in container", "Write below etc", "Sensitive file opened
  for reading", "Unexpected outbound connection", "Privilege escalation via setuid",
  "Package management launched in container", "Crypto miner signature". Pods like
  `ai-sentinel/prompt-svc-xxxx`. EVERY event `simulated: true`.
- Build it inside the existing `services/` Go module as `cmd/telemetry-generator` with its own
  `Dockerfile.telemetry`. Reuse `internal/httpx`.
- (Optional, nice-to-have) also publish `ai.runtime.observed` to NATS — not required for the SOC
  dashboard, which polls over HTTP.

## Frontend additions (frontend/)
Add top navigation across three pages: **Executive** (`/`, exists), **SOC** (`/soc`),
**Compliance** (`/compliance`). New BFF route handlers (server-side; container DNS):
- `GET /api/prompts` → `${PROMPT_SVC_URL}/prompts?limit=50`
- `GET /api/violations` → `${COMPLIANCE_SVC_URL}/compliance/violations?limit=100`
- `GET /api/controls?framework=` → `${COMPLIANCE_SVC_URL}/compliance/frameworks/{framework}/controls`
- `GET /api/runtime` → `${RUNTIME_SVC_URL}/runtime/events?limit=50` (default `http://telemetry-generator:8087`)
- Keep the explicit `TextDecoder('utf-8')` decode in `src/lib/services.ts` (UTF-8 fix from Phase 1).

### SOC Dashboard (`/soc`) — dark "threat" aesthetic, auto-refresh (poll every ~4s)
1. **Live Prompt Monitoring** — table/stream from `/api/prompts`: time, user, provider, decision
   badge (ALLOW/WARN/REDACT/BLOCK colors), risk score, categories. Newest streams in.
2. **DLP Alerts** — from `/api/violations`: framework, controlId, severity, time. REVIEW marker
   where applicable.
3. **Runtime Events (Falco/Tetragon)** — from `/api/runtime`: rule, priority color, pod/container,
   message. **Header MUST read "SIMULATED"** with a tooltip "No live kernel telemetry on the
   compose path (Brief R1)." Never imply live integration.

### Compliance Dashboard (`/compliance`)
- 8 framework score cards from `/api/frameworks`: displayName, score (ring/bar), controlCount,
  openViolations, and a **REVIEW PENDING** badge where `reviewRequired` is true.
- Click a framework → its controls (`/api/controls?framework=`): controlId, title,
  mappedCategories, and the verbatim REVIEW marker (regional frameworks show `TODO-VERIFY`).
- Violations table from `/api/violations`.
- Make the R5 posture explicit: a banner noting regional control IDs are placeholders pending
  product-owner verification.

## Labels & honesty (Brief R4/R5/R1/§12)
- Runtime feed and any synthetic telemetry: labelled **SIMULATED** in the UI.
- Never invent control IDs. Regional frameworks stay `TODO-VERIFY`.
- Detection stays real. No scripted detection of specific demo files.

## Verification Gate 2 (what I will record)
Per scenario (2,3,4,6): the trigger command, the detection categories + risk + decision (or the
discovery event for #6), the resulting violation rows (HIPAA for #3, etc.), and the dashboard
change. Plus SOC dashboard screenshot (with SIMULATED runtime feed) and Compliance dashboard
screenshot.
