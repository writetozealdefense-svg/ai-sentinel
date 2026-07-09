# Claude Code Build Brief — ZealDefense AI Sentinel™ (Phase 3 → Production Grade)

> **How to use this prompt.** Paste this whole file into Claude Code as the project brief for
> the repo at `D:\AISPM - Sentinel`. Work **phase by phase, top to bottom**. Do **not** start a
> phase until the previous phase's **Verification Gate** has passed with evidence recorded in
> `VERIFICATION.md`. Breadth-first scaffolding is a failure, not a deliverable.

---

## 0. Role & North Star

You are the engineering lead taking **ZealDefense AI Sentinel™** — an AI Security Posture
Management platform ("Discover. Govern. Secure. Every AI Interaction.") — from a gated MVP
(Phases 0–2 complete) to a **production-grade product demonstrable at Black Hat Europe**.

The audience is expert and adversarial: they will probe the demo. The single most important
property is **demonstrable authenticity** — every capability shown live must actually run, on
real (synthetic) inputs. "Production grade" here means: the demo path is bulletproof and
repeatable, the architecture path is genuinely hardened, and the engineering quality
(auth, observability, CI, tests, error handling) would survive a due-diligence review.

### Non-negotiable principles (carried forward from the MVP brief — still binding)
1. **Show, don't assert.** A component is "done" only when you have *run it* and pasted the
   observed output into `VERIFICATION.md`. Stubs reported as complete are unacceptable.
2. **Real detection on the demo path.** Never scripted if/else masquerading as detection.
3. **No invented facts.** Never fabricate compliance control IDs or framework clause numbers.
   Regional frameworks (NCA ECC, SAMA CSF, PDPL, DPDPA) keep their `TODO-VERIFY` placeholders
   until the product owner supplies verified IDs.
4. **Honest labelling.** Simulated feeds stay labelled `SIMULATED` in code, data, and UI.
   Real feeds (Helm-path Falco) are labelled as real only when they actually are.

---

## 1. Current State (verified — do not rebuild what exists)

Read `VERIFICATION.md`, `README.md`, and `docs/PHASE{1,2,3,4,5}-INTEGRATION.md` first.

**Gates passed with evidence:**
- **Gate 0** — contracts-first foundation: 8 OpenAPI 3.1 specs in `contracts/openapi/` (lint
  clean via `redocly.yaml` ruleset), NATS event schemas in `contracts/events/`, PostgreSQL DDL
  (`db/postgres/001_init.sql`, 10 tables), Neo4j model (`db/neo4j/schema.cypher`), generated Go
  stubs (`contracts/gen/go`, compiles) and TS client (`contracts/gen/ts`, typechecks).
- **Gate 1** — vertical slice: `customer_database.csv` → ChatGPT → real Presidio/YARA/regex
  detection (113 findings) → deterministic risk 100 → OPA `BLOCK` → violations from fixtures →
  live Executive Dashboard. Trigger: `scripts/scenario-1.sh`.
- **Gate 2** — breadth: Scenarios 2 (code+secrets→Claude), 3 (PHI→Gemini→HIPAA violation),
  4 (PCI→Copilot), 6 (DeepSeek shadow-AI) through the SAME pipeline; SOC dashboard (live prompt
  feed + `SIMULATED` Falco/Tetragon widget from `telemetry-generator`); Compliance dashboard
  (8 frameworks, fixture-driven, live scores).

**Running architecture (compose demo path, `deploy/compose/docker-compose.yml`):**
Go/Gin services in one module (`services/`): `discovery-svc` (8081→host 8091), `prompt-svc`
(8082), `compliance-svc` (8084), `telemetry-generator` (8087); Python FastAPI `detection-svc`
(8000, Presidio+YARA+regex); OPA (8181, policy `policy/opa/dlp.rego` + tests); Next.js 15
frontend (3000→host **3100**, BFF route handlers, pages `/`, `/soc`, `/compliance`);
datastores postgres/redis/nats(JetStream)/neo4j.

**Phase 3 code EXISTS and compiles/tests clean but is NOT yet integrated or gated:**
`services/cmd/agent-svc` (8083) + `internal/agentrisk` (tested), `services/cmd/graph-svc`
(8085, neo4j-go-driver v5, Postgres backfill + NATS live sync, 4 named Cypher queries),
`Dockerfile.agent`, `Dockerfile.graph`, `scripts/scenario-5.sh`, frontend `/agents` and
`/graph` pages (d3-force canvas) + BFF routes. Compose file already has `agent-svc` and
`graph-svc` service entries and frontend env (`AGENT_SVC_URL`, `GRAPH_SVC_URL`).
**A Docker Desktop crash interrupted the original Phase 3 verification** — image builds and
the live gate run were never completed. That is your first job.

**Environment facts (Windows 11 host):**
- Host ports 8081 and 3000 belong to OTHER projects (`kubeguard`, `blackhat-mfa`) — never bind
  them; sentinel uses host remaps 8091 (discovery) and 3100 (frontend). 8000/8082/8083/8084/
  8085/8087/8181 are sentinel's.
- Docker Desktop occasionally dies. Recovery: kill all `*docker*` processes, relaunch
  `C:\Program Files\Docker\Docker\Docker Desktop.exe`, poll `docker ps` (may take 2–4 min).
  If WSL wedges: `wsl --shutdown` first.
- `kind v0.23.0` is installed at `$(go env GOPATH)/bin/kind.exe`. Go 1.26, Node 24,
  Helm 3.21, kubectl 1.34 available. No live LLM API keys — Copilot must default offline (R3).
- Demo compliance fixtures: `fixtures/compliance/*.yaml`; demo datasets:
  `fixtures/demo-data/*`; every mapping carries `# REVIEW` markers (R5).

**Working method that is succeeding — keep it:** for each phase, first write/refresh the
binding integration contract (`docs/PHASE<N>-INTEGRATION.md`: exact ports, request/response
shapes, env vars, formulas), then fan out **multiple parallel agents** (one per directory:
`services/`, `frontend/`, `detection-svc/`, `deploy/`, `docs/`) that build against it and
self-verify by *running* their component and pasting evidence. The lead (you) NEVER delegates
integration or the gate run: wire compose/helm yourself, run the scenarios yourself, record
the gate yourself. Agents must not touch `deploy/compose/docker-compose.yml` or each other's
directories.

---

## 2. Phase 3 — Complete Agent Security + Risk Graph (finish what the crash interrupted)

Contract: `docs/PHASE3-INTEGRATION.md` (already written — follow it exactly).

1. Rebuild/verify: `cd services && go build ./... && go vet ./... && go test ./...`.
2. Build images (`Dockerfile.agent`, `Dockerfile.graph`), `docker compose build`, `up -d`.
3. Run `scripts/scenario-5.sh`: agent `finops-crew-01` (CrewAI) touches Jira → GitHub →
   Salesforce → sensitive data access → privilege escalation. Verify the score trajectory
   **12→24→36→51→76**, threshold-crossing at 70 emits **exactly one** violation row
   (`agent_id` set, control ID resolved from fixtures via category OTHER — never hardcoded),
   and re-running does NOT duplicate the violation.
4. Verify graph-svc backfilled Neo4j from Postgres and `POST /graph/query/phi-to-unauthorized-ai`
   returns **real rows** (nurse.joy@example.com → gemini → patient_records.csv from Scenario 3).
   Paste the Cypher and rows.
5. Screenshot `/agents` (risk gauge red at ≥70, THRESHOLD BREACH tag, SIMULATED labels) and
   `/graph` (animated force graph + verbatim Cypher + tabular rows).

> **Gate 3:** scenario-5 trajectory + single violation + idempotency proof; PHI graph query
> Cypher + real rows; both screenshots. Record in `VERIFICATION.md`.

---

## 3. Phase 4 — AI Security Copilot (offline RAG)

Contract: `docs/PHASE4-INTEGRATION.md` (already written — follow it exactly).
`copilot-svc` (Go, 8086): retrieval-grounded explanations of real incidents. **Offline
deterministic mode is the DEFAULT** — template composition over the evidence bundle
(violation row + linked prompt + findings + fixture control), three views (executive /
technical / compliance), `citations[]` on every answer. Live LLM ONLY behind
`COPILOT_LIVE=true` + key; never required. Frontend `/copilot` page: violations list →
Explain → three tabs + citations footer + mode badge `OFFLINE · DETERMINISTIC`.

> **Gate 4:** with no network and no keys, `GET /copilot/explain/{scenario-1 violationId}`
> returns all three views citing the real violation + fixture control (paste JSON); every fact
> in the text exists in the evidence bundle; UI screenshot; `/healthz` reports mode=offline.

---

## 4. Phase 5 — Kubernetes Hardening Path (the architecture story)

Contract: `docs/PHASE5-INTEGRATION.md` (already written — follow it exactly).
kind cluster `ai-sentinel`; umbrella Helm chart `deploy/helm/ai-sentinel` (all services,
fixtures/OPA via ConfigMap, Secrets for credentials, `imagePullPolicy: Never` + `kind load`);
namespace under **PSS restricted**; default-deny NetworkPolicies with explicit allows;
per-service ServiceAccounts; **real** Falco (`modern_ebpf`) + falcosidekick webhook →
telemetry-generator ingest mode (`RUNTIME_MODE=ingest`, events `simulated:false`, SOC badge
flips to `LIVE (Falco)`); Tetragon TracingPolicy; Kyverno enforce ClusterPolicy (disallow
privileged, require runAsNonRoot) demonstrated by a rejected `bad-pod.yaml`.
WSL2 kernel usually supports modern_ebpf; if eBPF genuinely fails on this machine, record the
observed error honestly and demonstrate Kyverno fully — do not fake runtime events.

> **Gate 5:** `helm lint` clean; `kubectl get pods -n ai-sentinel` all Ready; the Kyverno
> admission REJECTION output; a real Falco alert JSON (and its arrival in the SOC feed if
> sidekick wiring succeeds). Record honestly.

---

## 5. Phase 6 — Production-Grade Hardening (new scope beyond the MVP brief)

Split across parallel agents, integrated and verified by you:

**6a. AuthN/Z (Keycloak/OIDC — the stack's declared auth, currently absent).**
- Add Keycloak to compose (+ Helm), realm `ai-sentinel` provisioned from a committed realm
  export (demo users: `ciso@zealdefense.ai` admin, `analyst@zealdefense.ai` viewer).
- The Next.js BFF is the OIDC confidential client (Auth.js or equivalent): login page, session
  cookie, all