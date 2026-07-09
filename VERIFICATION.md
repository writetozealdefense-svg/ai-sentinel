# VERIFICATION.md — Evidence Log

> A component is "done" only when it has been **run** and the observed output pasted here.
> Stubs reported as complete are unacceptable (Brief §0, §12).

This file is the single source of truth for which Verification Gates have passed. Each gate
records: the exact command run, the observed output, and a PASS/FAIL verdict with date.

---

## Verification Gate 0 — Contracts & Datastores

**Requirements (Brief §3):**
- `docker compose up` brings the four datastores (Postgres, Redis, NATS, Neo4j) healthy.
- OpenAPI specs lint clean.
- Generated stubs compile.

### Evidence

Run date: 2026-06-21. Host: Windows 11, Docker 29.0.1 / Compose v2.40.3, Go 1.26.4, Node 24.14.1.

#### 0.1 — `docker compose ps` (four datastores healthy)

Command: `cd deploy/compose && docker compose up -d` then `docker compose ps`

```
NAME                IMAGE                COMMAND                  SERVICE    STATUS                    PORTS
sentinel-nats       nats:2.10-alpine     "docker-entrypoint.s…"   nats       Up 17 seconds (healthy)   0.0.0.0:4222->4222/tcp, 0.0.0.0:8222->8222/tcp
sentinel-neo4j      neo4j:5-community    "tini -g -- /startup…"   neo4j      Up 17 seconds (healthy)   0.0.0.0:7474->7474/tcp, 0.0.0.0:7687->7687/tcp
sentinel-postgres   postgres:16-alpine   "docker-entrypoint.s…"   postgres   Up 17 seconds (healthy)   0.0.0.0:5432->5432/tcp
sentinel-redis      redis:7-alpine       "docker-entrypoint.s…"   redis      Up 17 seconds (healthy)   0.0.0.0:6379->6379/tcp
```

All four report `(healthy)`. Functional verification beyond the healthcheck:

```
# Postgres — init DDL applied (psql \dt): 10 tables created
 agent_activity | agents | datasets | discovery_events | findings
 models | prompts | tools | users | violations   (10 rows)

# NATS — wget http://127.0.0.1:8222/healthz  →  {"status":"ok"}   (JetStream enabled via -js)

# Neo4j — cypher-shell 'RETURN 1 AS ok;'  →  ok / 1   (Bolt reachable, auth OK)
```

#### 0.2 — OpenAPI lint output

Command: `npx @redocly/cli@latest lint` (ruleset: recommended, tuned via `redocly.yaml`)

```
validating contracts/openapi/discover.yaml ... validated in 20ms
validating contracts/openapi/prompts.yaml ... validated in 12ms
validating contracts/openapi/dlp.yaml ... validated in 9ms
validating contracts/openapi/agents.yaml ... validated in 7ms
validating contracts/openapi/compliance.yaml ... validated in 10ms
validating contracts/openapi/graph.yaml ... validated in 9ms
validating contracts/openapi/copilot.yaml ... validated in 7ms
EXIT: 0
```

All 7 specs lint clean (0 errors, 0 warnings under the project ruleset). `security-defined`
is kept ON and satisfied by a declared Keycloak/OIDC scheme in every spec.

#### 0.3 — Generated stubs compile

**Go server stubs** — oapi-codegen v2.4.1 (`types,gin,spec`) for all 7 specs into
`contracts/gen/go/<spec>/`:

```
cd contracts/gen/go && go build ./...   →  BUILD EXIT: 0
                       go vet ./...      →  VET EXIT: 0
```
(Note: oapi-codegen emits a 3.1-support warning but generates compilable Go; domain types
e.g. `InspectRequest`, `InspectResult`, `Finding`, `Decision`, `ServerInterface` all present.)

**TypeScript client** — openapi-typescript v7 types for all 7 specs + `openapi-fetch`
typed clients in `contracts/gen/ts/`:

```
cd contracts/gen/ts && node generate.mjs   →  7 files generated
                       npx tsc --noEmit     →  TSC EXIT: 0
```

**Verdict:** ✅ PASS (2026-06-21)

---

## Verification Gate 1 — Vertical Slice (Scenario 1)

Run date: 2026-06-25. Cold `docker compose up` (4 datastores + detection-svc + opa +
discovery-svc + prompt-svc + compliance-svc + frontend). Trigger: `scripts/scenario-1.sh`
(and the dashboard "Run Scenario 1" button) — `customer_database.csv` → ChatGPT (unsanctioned).

> Host ports 8081 and 3000 were already taken by another project on this machine, so
> discovery-svc is published on **8091** and the frontend on **3100** (container ports
> unchanged: 8081 / 3000). All service-to-service wiring uses container DNS and is unaffected.

### 1.1 — detection-svc `/scan` response (REAL detection, no scripting)
`POST http://localhost:8000/scan` with the CSV:
```
stats: {"totalFindings": 113, "byCategory": {"PII": 100, "PCI": 10, "OTHER": 3}, "highestScore": 1.0, "durationMs": 99}
sample findings:
  {type: LOCATION,      category: PII, engine: presidio, start: 0,  end: 11, score: 0.85}
  {type: PERSON,        category: PII, engine: presidio, start: 58, end: 71, score: 0.85}
  {type: EMAIL_ADDRESS, category: PII, engine: presidio, start: 96, end:120, score: 1.0}
  {type: CREDIT_CARD,   category: PCI, engine: presidio, ...      score: 1.0}
```
Engines reported by `/healthz`: `{"presidio":27,"yara":12,"regex":8}`. YARA SECRET/CODE rules
verified firing on separate samples (AWS key, PEM private key, GitHub token; Python source).
Honest note: row-1 SSN `123-45-6789` is intentionally NOT flagged — Presidio correctly rejects
it as an invalid (checksum-failing) SSN. The other 9 SSNs are detected.

### 1.2 — OPA decision
`POST http://localhost:8181/v1/data/sentinel/dlp/decision`:
```json
{"result":{"action":"BLOCK","reasons":[
  "PCI data detected (10 finding(s)) sent to unsanctioned AI system chatgpt",
  "bulk PII (113) to unsanctioned AI system chatgpt"]}}
```

### 1.3 — prompt-svc orchestrated result (`InspectResult`)
`POST /prompts/inspect` →
```
promptId   : c4a44b45-1902-4e4d-b982-b6f47cb8fc96
decision   : BLOCK
riskScore  : 100        (severity 35 [PCI] + volume 40 + confidence 15 + unsanctioned 10, clamped)
violationId: f8473b6b-b702-4a1d-87fa-9a2822ed5b90   (returned synchronously)
policyTrace: PCI data detected (10 finding(s)) ...; bulk PII (113) to unsanctioned AI system chatgpt
redactedPreview: ...James Wilson,<EMAIL_ADDRESS> [REDACTED],123-45-6789,<CREDIT_CARD> [REDACTED],
                 "742 Evergreen Terrace, <LOCATION> [REDACTED], IL <US_ZIP_CODE> [REDACTED]"...
```

### 1.4 — Persisted rows (Postgres)
`prompts`: 1 row `BLOCK / risk 100 / scenario-1 / james.wilson@example.com / chatgpt`.
`findings`: aggregated rows persisted. `discovery_events`: `all_simulated = true` (Brief R4).
`violations` (one row per matched framework; control IDs **only** from fixtures — Brief R5):
```
 framework   |  control_id  | category | severity
-------------+--------------+----------+----------
 GDPR        | Art. 5(1)(f) | PII      | high
 ISO27001    | A.5.34       | PII      | high
 NIST_AI_RMF | GOVERN 1.1   | PII      | high
 PDPL        | TODO-VERIFY  | PII      | high     <- regional: placeholder preserved, not invented
 NCA_ECC     | TODO-VERIFY  | PII      | high     <- regional: placeholder preserved, not invented
 SAMA_CSF    | TODO-VERIFY  | PII      | high     <- regional: placeholder preserved, not invented
 DPDPA       | TODO-VERIFY  | PII      | high     <- regional: placeholder preserved, not invented
```
HIPAA correctly produced **0** violations (PHI-only framework; Scenario 1 is PII/PCI).

### 1.5 — Executive Dashboard (live, real data)
Screenshot captured (Next.js, dark "Black Hat" theme). Tiles observed:
- **AI Inventory**: `chatgpt · 3 uses · 1 user` with a red **SHADOW AI** badge and a `SIMULATED` tag.
- **Latest Inspection** (after clicking "Run Scenario 1"): red **BLOCK** badge, glowing **RISK 100**
  gauge, `FINDINGS BY CATEGORY: PII ×100 / PCI ×10 / OTHER ×3`, a **REDACTED PREVIEW** showing
  `<CREDIT_CARD> [REDACTED]` / `<US_SSN> [REDACTED]`, the **OPA policy trace**, and the
  promptId + violationId.
- **Open Violations by Framework**: Recharts bars across the 7 triggered frameworks with a
  **REVIEW PENDING** marker (fixtures still carry their `# REVIEW` markers).
- A frontend BFF UTF-8 mojibake bug (em dashes in framework display names) was found and fixed
  (`frontend/src/lib/services.ts` — decode upstream bytes with `TextDecoder('utf-8')`).

Every link in the chain is real (detection → risk → OPA → persistence → violation → dashboard);
none is faked.

**Verdict:** ✅ PASS (2026-06-25)

---

## Verification Gate 2 — Breadth (Scenarios 2,3,4,6 + SOC/Compliance dashboards)

Run date: 2026-06-25. Built with 3 parallel agents (data+detection, Go backend, frontend
dashboards) against `docs/PHASE2-INTEGRATION.md`; integrated + verified by me. All scenarios go
through the SAME `prompt-svc /prompts/inspect` — no bespoke per-scenario code paths. Stack
rebuilt clean (detection-svc PHI enhancements, prompt-svc `GET /prompts`, telemetry-generator,
frontend) before this run.

### 2.1 — Each scenario reproducible from a documented trigger (detection + risk delta + decision)
Triggers: `scripts/scenario-{2,3,4,6}.sh`. Observed against the live stack:
```
S2 code->claude    decision=BLOCK  risk=100  cats=[CODE, PII, SECRET]   (source_code_leak.py)
   policyTrace: SECRET data detected (6) ...; bulk PII (38) to unsanctioned AI system claude
S3 PHI->gemini     decision=BLOCK  risk=100  cats=[OTHER, PHI, PII]     (patient_records.csv)
   policyTrace: PHI data detected (47) ...; bulk PII (120) to unsanctioned AI system gemini
S4 PCI->copilot    decision=BLOCK  risk=100  cats=[OTHER, PCI, PII]     (payroll.csv)
   policyTrace: PCI data detected (20) ...; bulk PII (68) to unsanctioned AI system copilot
S6 deepseek shadow discovery_accepted=true | inspect decision=REDACT risk=52  (deepseek_prompt.txt)
```
Risk delta is real and category-driven: heavy CODE/PHI/PCI → BLOCK/100; light benign prompt →
REDACT/52. Detection is REAL — the data agent enhanced detection-svc PHI recognizers (inline MRN,
standalone ICD-10) and added an ABA routing-number regex so PHI/PCI genuinely fire on the new
datasets (validated against a throwaway detection-svc on an alt port before integration).

### 2.2 — Scenario 3 produces the HIPAA mapping (the "HIPAA shown" requirement)
The existing compliance mapping (no special-casing) turned the PHI finding into a HIPAA violation:
```
 framework |  control_id   | category | count
-----------+---------------+----------+-------
 HIPAA     | 164.312(e)(1) | PHI      |   2
```
HIPAA had 0 violations after Phase 1 (PII/PCI only); it appears now precisely because Scenario 3
introduced PHI. Control ID is from `fixtures/compliance/hipaa.yaml`, not invented.

### 2.3 — New backend surfaces
- `GET /prompts?limit=` (prompt-svc) returns newest-first summaries with derived categories — powers SOC live monitoring. Verified returning the S2/S3/S4/S6 inspections.
- `GET /runtime/events` (telemetry-generator, port 8087) returns SIMULATED Falco/Tetragon events; every event `simulated:true`. Verified.
- `GET /discover/inventory` now lists 5 shadow-AIs (chatgpt, deepseek, claude, copilot, gemini), all `sanctioned=false`.

### 2.4 — SOC Dashboard (screenshot captured, live data)
`/soc` — "SOC Threat Monitor · LIVE · AUTO-REFRESH 4S":
- **Live Prompt Monitoring**: streamed rows for all four scenarios with color-coded decision
  badges (BLOCK/REDACT), risk scores, provider, and category chips (CODE/PHI/PCI/SECRET/PII).
- **Runtime Events (Falco/Tetragon)**: panel header carries a prominent **SIMULATED** badge
  ("No live kernel telemetry on the compose path (Brief R1)"). Feed shows real polled events:
  "Privilege escalation via setuid" (tetragon/critical), "Terminal shell in container"
  (falco/warning), "Crypto miner signature detected", etc. — each with source, priority colour,
  pod·container, timestamp.
- **DLP Alerts**: live violations.

### 2.5 — Compliance Dashboard (screenshot captured, live data)
`/compliance` — "Compliance Posture":
- **R5 banner**: "Regional framework control IDs (NCA ECC, SAMA CSF, PDPL, DPDPA) are placeholders
  pending product-owner verification."
- 8 framework score cards. Scores are violation-driven (`100 − 10×openViolations`): **HIPAA 80/100
  (2 open)** from the PHI scenario; GDPR/ISO27001/NIST/regional driven to 0 by 14 open each. Every
  card shows a **REVIEW** badge (fixtures still carry `# REVIEW` markers); regional controls render
  `TODO-VERIFY`. Framework display names render correct UTF-8 (em dashes) — Phase 1 BFF fix holds.

**Verdict:** ✅ PASS (2026-06-25). Screenshot note: the preview screenshot tool times out on the
live-polling pages until network/animation are frozen; styled captures of `/soc` and `/compliance`
were obtained by freezing fetch+animations after data load. Page content/data verified both
visually and via the API layer above.

---

## Verification Gate 3 — Agent Security + Risk Graph (Scenario 5)

Run date: 2026-07-03. agent-svc/graph-svc/frontend code was written in the prior session against
`docs/PHASE3-INTEGRATION.md`; this session finished the integration after the Docker Desktop crash.
Integration fixes made (all verified below):
- **Host port remap**: another project now holds host 5432, so sentinel-postgres publishes on
  **15432** (container 5432 unchanged; in-cluster DSNs and `docker exec … psql` unaffected).
- **graph-svc panic fix** (`services/cmd/graph-svc/queries.go`): the subgraph pass passed a
  `(any, error)` callback returning `nil, nil` to `neo4j.ExecuteRead`; the driver does `result.(T)`
  which panics on a nil interface (500 on every `/graph/query/*`). Callback now returns a non-nil
  `struct{}{}`. Rebuilt + redeployed.
- **Stale frontend image**: the running image predated Phase 3 (`/agents`, `/graph`, BFF routes all
  404). Rebuilt from current source; also added `?q=<name>` deep-linking to `/graph` for the demo
  runbook.

### 3.1 — Scenario 5 score trajectory (real run, `scripts/scenario-5.sh`)
```
>> 1/5 tool_call jira (PROJ-1042)                    {"accepted":true,"agentId":"finops-crew-01","riskScore":12}
>> 2/5 tool_call github (payments-service)           {"accepted":true,"agentId":"finops-crew-01","riskScore":24}
>> 3/5 tool_call salesforce (accounts API)           {"accepted":true,"agentId":"finops-crew-01","riskScore":36}
>> 4/5 data_access salesforce.accounts (sensitive)   {"accepted":true,"agentId":"finops-crew-01","riskScore":51}
>> 5/5 privilege_escalation (github org admin)       {"accepted":true,"agentId":"finops-crew-01","riskScore":76}
```
Trajectory **12→24→36→51→76** exactly as the contract formula predicts (3×12 tools + 15 sensitive
data_access + 25 privilege_escalation = 76); threshold 70 crossed at event 5 only.
`GET /agents/finops-crew-01` → riskScore 76, tools `[jira, github, salesforce]`, riskFactors
`["3 tools (jira, github, salesforce)", "sensitive data access (salesforce.accounts)",
"privilege escalation"]`, `simulated: true` (Brief R4), full 5-event activity list.

### 3.2 — Exactly ONE violation, agent-linked, fixture-sourced (Brief R5)
```
                  id                  |  framework  | control_id | category | severity |    agent_id    | prompt_id
--------------------------------------+-------------+------------+----------+----------+----------------+-----------
 9c438128-554e-4459-85e8-0113ab1475a5 | NIST_AI_RMF | MAP 1.1    | OTHER    | low      | finops-crew-01 |
```
description: "Agent risk threshold breached (score 76): 3 tools (jira, github, salesforce);
sensitive data access (salesforce.accounts); privilege escalation". `prompt_id` NULL, `agent_id`
set. Control **MAP 1.1 / severity low comes from `fixtures/compliance/nist_ai_rmf.yaml`**
(mappedCategories `[OTHER]`) — not hardcoded.

### 3.3 — Idempotency (re-run of scenario-5.sh)
All five re-sent events returned `riskScore: 76` (score is recomputed from full history — distinct
tools / any-flags don't double-count) and
`SELECT count(*) FROM violations WHERE agent_id IS NOT NULL` → still **1** (violation fires on the
<70→≥70 crossing only, not on every event above threshold).

### 3.4 — Neo4j backfill + live NATS population (graph-svc)
Startup log after restart (agents arrived via backfill; during the first run they arrived live via
`ai.agent.activity` — both paths exercised):
```
[graph-svc] schema constraints applied (idempotent)
[graph-svc] backfill complete: map[agent-dataset edges:1 agent-tool edges:3 agents:1 models:5
                                   prompts:14 user-model edges:5 users:5 violations:101]
```
(Datasets are MERGEd inside the prompt/agent statements, hence no separate counter.)

### 3.5 — THE demo query returns REAL rows (`POST /graph/query/phi-to-unauthorized-ai`)
Executed Cypher (returned verbatim in the response, shown in the UI):
```cypher
MATCH (u:User)-[:GENERATED]->(p:Prompt)-[:USED]->(m:Model {sanctioned:false})
WHERE 'PHI' IN p.categories
OPTIONAL MATCH (p)-[:EXPOSED]->(d:Dataset)
RETURN u.email AS user, m.provider AS aiSystem, d.name AS dataset,
       p.riskScore AS riskScore, p.decision AS decision
```
Rows (real Scenario 3 data — it was run twice during Gate 2, hence two matching prompts):
```
user                   | aiSystem | dataset             | riskScore | decision
-----------------------+----------+---------------------+-----------+---------
nurse.joy@example.com  | gemini   | patient_records.csv | 100       | BLOCK
nurse.joy@example.com  | gemini   | patient_records.csv | 100       | BLOCK
```
Response also carries the renderable subgraph: 5 nodes (User, 2× Prompt with
`categories:[OTHER,PHI,PII]`, Model gemini `sanctioned:false`, Dataset patient_records.csv) and
6 edges (GENERATED/USED/EXPOSED). Other named queries verified live: `high-risk-agents` →
finops-crew-01/76 with its 3 tools; `blocked-prompts-by-user` → 12 rows; `full-graph` →
135 nodes · 151 edges (101 Violation, 14 Prompt, 6 Dataset, 5 Model, 5 User, 3 Tool, 1 Agent).

### 3.6 — Dashboards (screenshots in `docs/screenshots/`, live data, headless-Edge capture)
- `phase3-agents.png` — `/agents`: FinOps Automation Crew card with CREWAI badge, owner, tool
  chips, red risk gauge **76**, **THRESHOLD BREACH** badge, the three riskFactors, **SIMULATED**
  tag, and footer "AGENT TELEMETRY IS SIMULATED (BRIEF R4) · RISK THRESHOLD = 70".
- `phase3-graph-full.png` — `/graph` default full-graph: animated d3-force canvas,
  "135 NODES · 151 EDGES", color-coded legend (User/Prompt/Model/Agent/Tool/Dataset/Violation),
  FinOps Automation Crew agent node visible, executed Cypher panel + label-count result rows.
- `phase3-graph-phi.png` — `/graph?q=phi-to-unauthorized-ai`: the highlighted 5-node/6-edge PHI
  subgraph (nurse.joy@example.com → prompts → gemini → patient_records.csv), the verbatim contract
  Cypher, and the 2 result rows. Footer: "GRAPH DATA LIVE FROM GRAPH-SVC (NEO4J) VIA BFF".

**Verdict:** ✅ PASS (2026-07-03)

---

## Verification Gate 4 — AI Security Copilot (offline RAG)

Run date: 2026-07-03. Built with 2 parallel agents (Go copilot-svc; frontend /copilot + BFF)
against `docs/PHASE4-INTEGRATION.md`; both self-verified (go build/vet/test + local smoke against
the live DB; `next build` + lint clean). Integrated + gated by the lead. One integration bug found
and fixed: the deterministic ask-router only matched the literal word order "riskiest agent", so
the UI's suggested question "Which agent is the riskiest?" fell through to the fallback —
`internal/copilot/router.go` now matches risk-word + subject co-occurrence (unit tests extended
with the UI phrasings; `go test ./internal/copilot/` green).

### 4.1 — Offline is the default (no network, no keys)
Container env (full): `COMPLIANCE_FIXTURES=/fixtures/compliance`, `COPILOT_LIVE=false`,
`DATABASE_URL=…` — **no COPILOT_API_KEY anywhere**. `GET /healthz` →
`{"mode":"offline","service":"copilot-svc","status":"ok"}`.

### 4.2 — `GET /copilot/explain/{scenario-1 violationId}` (violation 5d1c071c…, GDPR Art. 5(1)(f))
Returns `mode: "offline"` and **all three views**, every fact retrieved from the platform's own
data (trimmed; full JSON reproducible via
`curl http://localhost:8086/copilot/explain/5d1c071c-b6dd-4ae9-be1d-d1d1eddc91fa`):
- **executive**: "james.wilson@example.com submitted content to chatgpt, an unsanctioned AI
  system, and the platform decision was BLOCK. The artifact involved was
  \"customer_database.csv\" … high-severity violation under EU General Data Protection Regulation
  (control Art. 5(1)(f): Integrity and confidentiality). Prompt risk score: 100/100 …"
- **technical**: the REAL findings ("CREDIT_CARD (PCI) x10 via presidio, score 1.00; …
  US_SSN (PII) x9 …"), risk breakdown "severity 35 + volume 40 + confidence 15 + unsanctioned 10
  = 100/100; stored prompt risk score 100/100" (recomputed via internal/risk and cross-checked
  against the stored score), the real OPA policy trace, and a 200-char redaction sample
  ("<CREDIT_CARD> [REDACTED]" …).
- **compliance**: the fixture control quoted verbatim — "GDPR control Art. 5(1)(f) — 'Integrity
  and confidentiality': Personal data processed with appropriate security…", mapped categories
  PII, the fixture's REVIEW marker surfaced verbatim ("# REVIEW: verify control ID"), remediation
  posture line.
- **citations (15)**: kind=violation (ref 5d1c071c…), kind=control (ref "Art. 5(1)(f)"),
  kind=prompt (ref ed79f460…), 12× kind=finding (real finding row UUIDs). UTF-8 integrity
  verified at the byte level (em dashes intact; earlier mojibake was a terminal-display artifact).

### 4.3 — `POST /copilot/ask` deterministic router (all against the deployed container)
- "why was the last prompt blocked" + view=executive → grounded on the most recent BLOCK
  violation (hr.admin@example.com → copilot, payroll.csv), views filtered to executive only,
  13 citations.
- "Which agent is the riskiest?" → "FinOps Automation Crew (finops-crew-01) … risk score of
  76/100 … Risk factors: 3 tools (jira, github, salesforce); sensitive data access
  (salesforce.accounts); privilege escalation." (cites the Phase-3 threshold violation).
- "Who is the riskiest user?" → james.wilson@example.com, 42 recorded violations, 3 citations.
- "What is our compliance posture?" → real per-framework open counts (DPDPA 14; GDPR 14; HIPAA 2;
  ISO27001 14; NCA_ECC 14; NIST_AI_RMF 15; PDPL 14; SAMA_CSF 14), 8 citations.
- "what is the weather in dubai" → honest capability-list fallback, `citations: []` — never
  hallucinates.
- Unknown UUID → 404 `{"error":"not found","detail":"no violation with id …"}`; non-UUID → 400.

### 4.4 — Live-mode gate (not exercised, by design)
`NewLiveClient` returns nil unless `COPILOT_LIVE=true` AND `COPILOT_API_KEY` are both set — the
live path is structurally unreachable in this deployment; covered by unit tests, not network
calls. On any live-path error the service falls back to the offline composition and reports
`mode:"offline"` honestly.

### 4.5 — `/copilot` UI (screenshots in `docs/screenshots/`, live data)
`phase4-copilot-{executive,technical,compliance}.png` — captured via
`/copilot?explain=5d1c071c…&tab=…` (deep-link added for the demo runbook): header badge
**OFFLINE · DETERMINISTIC**, Recent Violations list (topped by the Phase-3 agent violation),
suggested-question chips, the three tabs each rendering their grounded narrative, and a
"GROUNDED IN" citations footer showing the violation/control/prompt chips + 12 finding chips
with real row UUIDs. Footer: "ANSWERS ARE GROUNDED IN STORED VIOLATIONS + FIXTURE CONTROLS ·
OFFLINE DETERMINISTIC MODE IS THE DEFAULT (BRIEF R3)".

**Verdict:** ✅ PASS (2026-07-03)

---

## Verification Gate 5 — Kubernetes Hardening Path

Run completed 2026-07-08 on the recovered machine (after the 2026-07-04 Docker Desktop failure
below). The `kind` cluster survived in its VHDX; the stack was repaired and the gate run in full.
**Verdict: ✅ PASS (with honestly-recorded eBPF environment limits).**

### 5.G — Gate evidence (live run)

**5.G.1 — Full stack Ready on the Helm path.** `helm install ai-sentinel` + Kyverno + Falco +
Tetragon. `kubectl get pods -n ai-sentinel` — all **14/14 Running 1/1**
(agent/compliance/copilot/detection/discovery/frontend/graph/nats/neo4j/opa/postgres/prompt/redis/
telemetry-generator); kyverno ns 4/4; falco ns 2/2 (falco + falcosidekick); tetragon DaemonSet
Ready. `helm lint deploy/helm/ai-sentinel` → `1 chart(s) linted, 0 chart(s) failed`.

Four real bugs were found and fixed to get pods healthy (all committed to the chart/values):
- **postgres** CrashLoopBackOff — the liveness probe killed initdb on this slow VM disk before it
  finished; combined with the below it looped. Fixed by the enableServiceLinks change + clean
  re-init.
- **neo4j** fatal `Unrecognized setting: PORT.7687.TCP.PORT` — Kubernetes service-link env vars
  (a Service named `neo4j` injects `NEO4J_PORT_*`) that the neo4j entrypoint misparses as config.
  Fixed by adding `enableServiceLinks: false` to all 14 pod specs.
- **opa** `exec /opa: exec format error` — the wedged in-node `:latest` pull (2026-07-04) left a
  truncated layer that containerd kept reusing via the existing snapshot chain. Fixed by purging
  the corrupt snapshot chain in the node and pinning `openpolicyagent/opa:1.18.2` (also §3.2
  hygiene). Verified the re-unpacked `/opa` is a real 79 MB ELF (`7f 45 4c 46`).
- **falco** saw no workload-pod events — the chart's DaemonSet omits `hostPID`; patched
  `hostPID: true` (documented in falco-values, applied via `kubectl patch` since the chart exposes
  no value).

**5.G.2 — Kyverno admission rejection (bad-pod), verbatim.** Both Enforce ClusterPolicies show
`ADMISSION true / READY True`. `kubectl apply -f deploy/k8s-tests/bad-pod.yaml`:
```
Error from server: error when creating "deploy/k8s-tests/bad-pod.yaml": admission webhook
"validate.kyverno.svc-fail" denied the request:

resource Pod/ai-sentinel/bad-pod was blocked due to the following policies

disallow-privileged-containers:
  privileged-containers: 'validation error: Privileged mode is disallowed in the ai-sentinel
    namespace. The fields spec.containers[*].securityContext.privileged ... must be unset or
    set to `false`. rule privileged-containers failed at path /spec/containers/0/securityContext/privileged/'
require-run-as-nonroot:
  run-as-non-root: 'validation error: Running as root is not allowed in the ai-sentinel
    namespace. ... rule run-as-non-root[0] failed at path /spec/securityContext/runAsNonRoot/
    rule run-as-non-root[1] failed at path /spec/containers/0/securityContext/runAsNonRoot/'
```
(Both the Kyverno webhook AND the built-in PSS-restricted label reject it — defence in depth; the
PSS enforce was momentarily set to `privileged` only to let the request reach the Kyverno webhook
for this capture, then restored to `restricted`.)

**5.G.3 — REAL Falco alert (simulated:false) → SOC LIVE feed.** Falco loaded the **modern BPF
probe** (`Opening 'syscall' source with modern BPF probe`). A `cat /etc/shadow` produced a genuine
alert with real file path + process lineage:
```
{"hostname":"ai-sentinel-control-plane","output":"07:35:49.106322229: Warning Sensitive file
opened for reading by non-trusted program | file=/etc/shadow ... process=cat
proc_exepath=/usr/bin/cat parent=sh command=cat /etc/shadow ...","priority":"Warning",
"rule":"Read sensitive file untrusted","source":"syscall", ...}
```
falcosidekick forwarded it — `Webhook - POST OK (202)` — to
`http://telemetry-generator.ai-sentinel.svc:8087/runtime/events` (RUNTIME_MODE=ingest), which
normalized it and served it from `GET /runtime/events`:
```
eventId: 547cd588-... | source: falco | rule: Read sensitive file untrusted
priority: warning | simulated: False
```
The SOC dashboard Runtime Events panel then rendered the green **LIVE (Falco)** badge with the
real FALCO-tagged WARNING events (screenshot `docs/screenshots/phase5-soc-live-falco.png`,
captured against the Helm frontend via port-forward; `/api/runtime` served 4 events,
`all simulated false: True`, `sources: {falco}`).

**5.G.4 — Tetragon.** DaemonSet Ready; process_exec lineage events flow. The
`file-monitoring-etc-shadow` TracingPolicy required two authoring fixes to load
(`returnArgAction: Post` is unsupported in this Tetragon version → removed the return probe; the
`index 1 == "4"` MAY_READ mask silently dropped reads → path-only selector). After the fixes the
kprobe on `security_file_permission` **loads successfully** (BTF present, "Loaded sensor
successfully").

### 5.L — Honest environment limits (eBPF under nested WSL2 → kind; per the contract)
- **Falco attributes only node-root-namespace syscalls.** `kubectl exec` INTO a workload pod does
  NOT fire the container-context rules — falco logs `disabled BPF iterators (not running in the
  root PID namespace)` because the falco pod (inside the kind node container, inside the WSL2 VM)
  is not in the WSL2 kernel's true root PID namespace, even with hostPID. Syscalls in the node's
  own namespace (e.g. `docker exec node cat /etc/shadow`) ARE captured with full context — which
  is what produced the real alert above. Real detection is proven; nested-pod attribution is the
  known limit.
- **Tetragon's `security_file_permission` kprobe loads but emits no events** on /etc/shadow reads
  in this nested environment, though its tracepoint-based process_exec sensor works. Same eBPF
  nesting limit. No events were fabricated (contract R1 / Phase-5 honesty clause).
- **SOC panel static text.** The Runtime Events badge correctly flips to **LIVE (Falco)**, but the
  panel subtitle ("Events are synthetic") and page footer ("RUNTIME FEED IS SIMULATED") are static
  strings the SOC page did not make conditional — cosmetic inconsistency, flagged for Phase 6
  frontend polish. The contractually-specified signal (the badge) is correct.

**Verdict:** ✅ PASS (2026-07-08) — all required gate items met with real evidence (pods Ready,
Kyverno verbatim rejection, real Falco alert reaching the SOC LIVE feed with simulated:false,
helm lint clean). eBPF pod-namespace attribution limits recorded honestly, not worked around.

---

### Prior interruption record (2026-07-04)
Below: all Phase 5 artifacts were authored and offline-validated; the first live gate run was
interrupted by a machine-level Docker Desktop failure.

### Authored + validated so far (3 parallel agents; evidence in their build reports)
- **Helm umbrella chart** `deploy/helm/ai-sentinel/` — 14 components (Deployment+Service+SA each),
  fixtures/OPA-policy/postgres-init ConfigMaps, Secret-backed passwords, default-deny
  NetworkPolicies + 9 allows, PSS-restricted pod specs (documented exceptions: detection-svc and
  neo4j need writable root FS). `helm lint`: 0 failed. `helm template`: 56 objects render.
- **Runtime ingest mode** — telemetry-generator `RUNTIME_MODE=ingest` accepts falcosidekick
  webhooks, normalizes to RuntimeEvent with `simulated:false` (`simulate` default byte-identical,
  POST → 409); `contracts/openapi/runtime.yaml` + new `runtime.observed.schema.json` updated;
  SOC badge flips to **LIVE (Falco)** when all events are real. go build/vet/test green; Redocly
  clean; `next build` clean.
- **Runtime security add-ons** `deploy/helm/runtime-security/` — falco-values (modern_ebpf +
  sidekick webhook → telemetry-generator, verified by rendering the real upstream chart),
  tetragon values + /etc/shadow TracingPolicy (validated against the upstream CRD),
  Kyverno ClusterPolicies (Enforce, scoped to ns ai-sentinel; kubeconform-valid),
  `deploy/k8s-tests/bad-pod.yaml`, install runbook README.
- **kind cluster**: first bring-up hit kubeadm's 4m api-check deadline — the API server on this
  machine needs ~4.5 min to first become healthy (slow etcd fsync; verified: the retained node's
  apiserver reached `healthz: ok` shortly after the deadline). Fixed via
  `deploy/helm/kind-cluster.yaml` (`timeoutForControlPlane: 20m0s`); cluster then created Ready.
  All 9 app images `kind load`-ed; `helm install` succeeded; namespace labeled PSS restricted;
  postgres/nats + all app pods reached Running/1/1 progressively.
- **Environment failure (honest)**: during the datastore image pulls *inside* the kind node,
  Docker Desktop's WSL VM wedged repeatedly (docker CLI hangs, kind API TLS timeouts) and finally
  died with the `docker-desktop` WSL distro unregistered; 6 recovery cycles over ~70 min failed.
  Machine reboot required. Data intact (`docker_data.vhdx`). Prevention adopted for the re-run:
  pre-pull ALL upstream images on the host and `kind load` them (no in-node pulls), keep the
  compose stack stopped during cluster work.

---

## Verification Gate 6 — Conference Hardening (cold-clone 5-min script)

**Status:** NOT STARTED
