# Phase 1 Integration Contract (Scenario 1)

This is the binding contract for the Phase 1 vertical slice. Every component is built against
the interfaces below so independently-built parts integrate without rework. **Do not deviate.**

## Scenario 1 (the one path that must work end-to-end)
Employee `james.wilson@example.com` uploads `customer_database.csv` to **ChatGPT** (unsanctioned)
→ Discovery records the AI usage → Prompt Inspection calls `detection-svc` (REAL Presidio/regex/YARA)
→ PII detected → risk score computed → OPA returns **BLOCK** + redaction preview → a compliance
violation is emitted from fixtures → the event is visible live on the Executive Dashboard.

## Services, ports, container names (compose path)
| Component        | Container        | Internal URL                 | Host port |
|------------------|------------------|------------------------------|-----------|
| detection-svc    | `detection-svc`  | http://detection-svc:8000    | 8000      |
| discovery-svc    | `discovery-svc`  | http://discovery-svc:8081    | 8081      |
| prompt-svc       | `prompt-svc`     | http://prompt-svc:8082       | 8082      |
| compliance-svc   | `compliance-svc` | http://compliance-svc:8084   | 8084      |
| opa              | `opa`            | http://opa:8181              | 8181      |
| frontend         | `frontend`       | http://frontend:3000         | 3000      |
| postgres/redis/nats/neo4j | (Phase 0) | postgres:5432 / redis:6379 / nats:4222 / neo4j:7687 | — |

## Environment variables (consumed by services)
```
DATABASE_URL=postgres://sentinel:sentinel_dev_pw@postgres:5432/sentinel?sslmode=disable
NATS_URL=nats://nats:4222
DETECTION_URL=http://detection-svc:8000
OPA_URL=http://opa:8181
REDIS_URL=redis://redis:6379
COMPLIANCE_FIXTURES=/fixtures/compliance      # mounted read-only into the Go services
```

## Interface 1 — detection-svc  (Python FastAPI; contract = contracts/openapi/dlp.yaml)
`POST /scan`  body: `{"content": "<raw text>", "contentType": "csv", "filename": "customer_database.csv"}`
Response `200`:
```json
{
  "findings": [
    {"type":"US_SSN","category":"PII","engine":"presidio","text":"123-45-6789","start":0,"end":11,"score":0.95,"ruleName":null}
  ],
  "redactedPreview": "... <US_SSN> [REDACTED] ...",
  "stats": {"totalFindings": 40, "byCategory": {"PII": 40}, "highestScore": 0.99, "durationMs": 12}
}
```
- Categories: `PII | PHI | PCI | SECRET | CODE | OTHER`. Engines: `presidio | yara | regex`.
- Redaction format is exactly `<TYPE> [REDACTED]` (e.g. `<US_SSN> [REDACTED]`).
- `GET /healthz` → `{"status":"ok","engines":{"presidio":N,"yara":N,"regex":N}}`.

## Interface 2 — OPA decision  (Rego; served by `opa run --server`)
Query: `POST http://opa:8181/v1/data/sentinel/dlp/decision` body `{"input": <below>}`.
**Input:**
```json
{
  "provider": "chatgpt",
  "sanctioned": false,
  "categories": ["PII"],
  "totalFindings": 40,
  "highestScore": 0.99,
  "byCategory": {"PII": 40}
}
```
**Output** (`result`): `{"action":"BLOCK","reasons":["bulk PII (40) to unsanctioned AI system chatgpt"]}`
**Decision rules** (precedence BLOCK > REDACT > WARN > ALLOW; package `sentinel.dlp`, rule `decision`):
- `BLOCK` if any of `SECRET`, `PHI`, `PCI` in categories; OR (`PII` in categories AND totalFindings >= 5)
- `REDACT` if `PII` in categories AND totalFindings < 5
- `WARN` if `CODE` in categories (and not blocked/redacted)
- `ALLOW` otherwise
- Always include human-readable `reasons` (array of strings).

## Interface 3 — prompt-svc  (Go/Gin; contract = contracts/openapi/prompts.yaml)
`POST /prompts/inspect` orchestrates the whole path:
1. call `detection-svc /scan` with the content,
2. compute **risk score** (formula below),
3. call OPA `/v1/data/sentinel/dlp/decision`,
4. persist `prompts` + `findings` rows (Postgres),
5. publish `ai.prompt.inspected` to NATS,
6. if decision ∈ {REDACT, BLOCK} publish `ai.violation.raised` per matched framework (see compliance),
7. return `InspectResult`.

**Risk score formula (deterministic, 0–100):**
```
severityWeight = max over categories of {SECRET:40, PHI:35, PCI:35, PII:25, CODE:20, OTHER:5}
volume         = min(totalFindings, 20) * 2          # cap 40
confidence     = round(highestScore * 15)            # 0..15
unsanctioned   = sanctioned ? 0 : 10
score          = clamp(severityWeight + volume + confidence + unsanctioned, 0, 100)
```
For Scenario 1 this yields a HIGH score (~78) and decision `BLOCK`.

## Interface 4 — compliance-svc  (Go/Gin; contract = contracts/openapi/compliance.yaml)
- On startup loads `fixtures/compliance/*.yaml` (Brief R5 — control IDs ONLY from fixtures).
- Consumes `ai.violation.raised`; persists `violations` rows; serves `/compliance/*`.
- **Mapping rule:** for the finding categories, for EACH framework fixture, select controls whose
  `mappedCategories` intersect the categories; emit ONE violation per framework using that
  framework's highest-severity matching control. (Scenario 1 = PII → GDPR, ISO27001, NIST_AI_RMF,
  PDPL, NCA_ECC, SAMA_CSF, DPDPA each get one violation row.)
- `reviewRequired: true` on any framework whose matched control still carries its `# REVIEW` marker.

## Interface 5 — discovery-svc  (Go/Gin; contract = contracts/openapi/discover.yaml)
- `POST /discover/events` records a SIMULATED discovery event (Postgres `discovery_events`,
  publish `ai.discovery.observed`). prompt-svc fires this at the start of inspection.
- `GET /discover/inventory` aggregates by provider for the dashboard.

## Interface 6 — frontend Executive Dashboard  (Next.js 15)
- Uses Next.js **route handlers as a BFF** (server-side calls to `prompt-svc`/`compliance-svc`/
  `discovery-svc` by container DNS) so the browser has no CORS issues.
- A "Run Scenario 1" control posts the bundled `customer_database.csv` content to
  `prompt-svc /prompts/inspect` (provider=chatgpt, userEmail=james.wilson@example.com,
  scenario="scenario-1") and renders the returned decision/risk/findings live.
- Tiles: AI inventory, latest inspections (decision + risk badge), open violations by framework.
- Dark "Black Hat" aesthetic. Anything synthetic is labelled `SIMULATED`.

## NATS subjects (from contracts/events/SUBJECTS.md)
`ai.discovery.observed`, `ai.prompt.inspected`, `ai.violation.raised` — validate payloads against
the JSON Schemas in `contracts/events/`.

## Postgres tables (from db/postgres/001_init.sql) — already created
`users, models, discovery_events, prompts, findings, datasets, agents, tools, agent_activity, violations`.

## Trigger of record
`scripts/scenario-1.sh` reads `fixtures/demo-data/customer_database.csv` and `curl`s
`prompt-svc /prompts/inspect`. The dashboard button does the same via the BFF.
