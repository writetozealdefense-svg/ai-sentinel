# Phase 4 Integration Contract (AI Security Copilot — offline RAG)

Phases 0–3 gated. This phase adds `copilot-svc` and a Copilot UI panel. **Deterministic
OFFLINE mode is the DEFAULT (Brief R3)** — fully functional with no network and no API keys.
Live LLM only behind `COPILOT_LIVE=true` + `COPILOT_API_KEY`.

## Service, port
| Component   | Container     | Internal URL              | Host port |
|-------------|---------------|---------------------------|-----------|
| copilot-svc | `copilot-svc` | http://copilot-svc:8086   | 8086      |

Env: `DATABASE_URL`, `COMPLIANCE_FIXTURES=/fixtures/compliance`, `COPILOT_LIVE` (default
false/absent), `COPILOT_API_KEY` (optional), `COPILOT_MODEL` (optional).

## Interface — copilot-svc (Go/Gin, port 8086; contract = contracts/openapi/copilot.yaml)
### Grounding = retrieval, not generation
This is RAG over the platform's OWN data. For a given violation/incident the service retrieves:
1. the `violations` row (framework, control_id, severity, category, description, user, time),
2. the linked `prompts` row (provider, decision, risk_score, policy_trace, redacted_preview,
   scenario, artifact),
3. the aggregated `findings` for that prompt (types, categories, engines, counts),
4. the matching fixture control (title, description, mappedCategories, review marker) via
   `internal/compliance` — NEVER invent control text,
5. (agent violations) the `agents` row + risk factors.
Retrieval result = the **evidence bundle**. Every response cites it (`citations[]` per the
OpenAPI schema: kind violation|control|prompt|finding + ref + excerpt).

### Offline mode (DEFAULT)
Deterministic template composition over the evidence bundle — three views, each a short
grounded narrative built ONLY from retrieved fields (no free invention):
- **executive**: business risk framing — who, what data category, to which AI system, decision
  taken, compliance exposure (framework + control title), severity.
- **technical**: pipeline mechanics — detection engines + entity types/counts, risk-score
  breakdown (severity/volume/confidence/unsanctioned per the Phase-1 formula), OPA policy trace,
  redaction sample (first ~200 chars of redacted_preview).
- **compliance**: the fixture control quoted (framework, controlId, title, description),
  mapped categories, the REVIEW marker surfaced verbatim, and remediation posture
  (block recorded, violation logged, user coaching recommended).
`/healthz` reports `{"status":"ok","mode":"offline"}` unless live is enabled AND keyed.

### Endpoints
- `GET /copilot/explain/{violationId}` → AskResponse `{mode, views{executive,technical,
  compliance}, citations[]}` (404 if unknown).
- `POST /copilot/ask` `{question, violationId?, view?}`:
  - with violationId → same grounding; view=all|executive|technical|compliance filters views.
  - without violationId → intent match the question against a small deterministic router:
    "why was this/the last prompt blocked" → most recent BLOCK violation; "riskiest user/agent" →
    aggregate from DB; "compliance posture" → framework scores summary. Unknown → honest
    fallback listing what the copilot CAN answer (never hallucinate).
- Live mode (`COPILOT_LIVE=true` + key): same evidence bundle sent to the LLM with instructions
  to answer ONLY from it; mode reported "live". Keep the client minimal (Claude Messages API);
  do NOT verify live mode in tests (no key in this environment) — unit-test that without the
  flag the code path is unreachable.

## Frontend
Nav gains **Copilot** (`/copilot`). BFF: `GET /api/copilot/explain/[id]` and
`POST /api/copilot/ask` → `${COPILOT_SVC_URL:-http://copilot-svc:8086}`.
Page: chat-style panel. Left: recent violations list (from `/api/violations`) — click one →
"Explain" → renders the three views as tabs (Executive/Technical/Compliance) with a citations
footer (chips: violation/control/prompt/finding refs). Free-text ask box with suggested
questions. A visible mode badge: **OFFLINE · DETERMINISTIC** (or LIVE). Empty/error states,
no mocked data.

## Verification Gate 4 (what I record)
With `COPILOT_LIVE` unset and no keys: `GET /copilot/explain/{scenario-1 violationId}` returns
all three views, each citing the real violation row + fixture control (paste the JSON). The
answer text must contain ONLY facts present in the evidence bundle. UI screenshot of the three
tabs. `/healthz` shows mode offline.
