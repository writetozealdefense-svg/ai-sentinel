# Claude Code Build Prompt — ZealDefense AI Sentinel™ (Production Grade, Phase 3 → 6)

> **How to use.** Paste this whole file into a fresh Claude Code session opened at the repo root
> `D:\AISPM - Sentinel`. It is self-contained: it tells you exactly what already exists, the
> method that has been working, and what "production grade" means for the phases that remain.
> Work **phase by phase**. Do **not** start a phase until the prior **Verification Gate** has
> passed and its evidence is pasted into `VERIFICATION.md`. Breadth-first scaffolding is a
> failure, not a deliverable.

---

## 0. Who you are and the prime directive

You are the engineering lead completing a **conference-demo-ready, production-grade MVP** of an
enterprise Shadow-AI discovery, governance, DLP, agent-security and compliance platform:

**ZealDefense AI Sentinel™ — *"Discover. Govern. Secure. Every AI Interaction."***

Target audience: expert attendees at **Black Hat Europe** (and GCC CISO / healthcare / government
events). They will probe the demo live. The single most important property is **demonstrable
authenticity**: every capability shown must actually run, on real (synthetic) inputs, in front of
people who can tell the difference.

### Non-negotiable principles (inherited from the original brief — still binding)
1. **Show, don't assert.** A component is "done" only when you have *run it* and pasted the
   observed output into `VERIFICATION.md`. Stubs reported as complete are unacceptable.
2. **Real detection on the demo path.** PII/PHI/PCI/secret/code detection is genuinely functional
   (Presidio + YARA + regex), never scripted if/else.
3. **No invented facts.** Never fabricate compliance control IDs or framework clause numbers.
   Where ground truth is missing leave `// TODO: VERIFY` / `TODO-VERIFY` and surface it.
4. **Honest labelling.** Every simulated feed is labelled `SIMULATED` in code, data, and UI.
   Never imply integrations the platform lacks.

---

## 1. CURRENT STATE — what already exists (do not rebuild it)

Repo root: `D:\AISPM - Sentinel` (Windows; PowerShell primary, Bash available; **not** a git repo
yet). Fixed stack: Next.js 15 + TS + Tailwind + ShadCN + Recharts · Go 1.26 + Gin · Python 3.11
FastAPI (`detection-svc`) · Postgres/Redis/Neo4j/NATS · OPA · (Keycloak/OIDC declared in contracts)
· Prometheus/Grafana/OTel planned · Falco/Tetragon/Kyverno on the K8s path · Docker/K8s/Helm.

### Phases already COMPLETE and gated (evidence in `VERIFICATION.md`)
- **Gate 0 — Contracts & datastores: PASS.** 7 OpenAPI 3.1 specs in `contracts/openapi/`
  (+ `runtime.yaml` added in P2), NATS subjects + JSON Schemas in `contracts/events/`, Postgres DDL
  `db/postgres/001_init.sql` (10 tables), Neo4j model `db/neo4j/schema.cypher`, generated Go stubs
  (`contracts/gen/go`) and TS client (`contracts/gen/ts`) both compile, Redocly lint clean via
  `redocly.yaml`. Compose datastores healthy.
- **Gate 1 — Vertical slice (Scenario 1): PASS.** `customer_database.csv` → ChatGPT → discovery →
  detection (real, ~113 findings) → risk 100 → OPA BLOCK → 7 framework violations → Executive
  Dashboard. Trigger `scripts/scenario-1.sh`.
- **Gate 2 — Breadth (Scenarios 2,3,4,6) + SOC & Compliance dashboards: PASS.** S2 code→Claude
  BLOCK, S3 PHI→Gemini BLOCK + **HIPAA** violation, S4 PCI→Copilot BLOCK, S6 DeepSeek shadow-AI +
  REDACT/52 (real risk delta). SOC dashboard (live prompt feed + DLP alerts + SIMULATED runtime
  widget), Compliance dashboard (8 frameworks, R5 review banner). Scripts `scripts/scenario-{2,3,4,6}.sh`.

### Services already built (in `services/` — one Go module `github.com/zealdefense/ai-sentinel/services`)
`cmd/`: discovery-svc(8081→host 8091), prompt-svc(8082), compliance-svc(8084),
telemetry-generator(8087), **agent-svc(8083)**, **graph-svc(8085)**. Shared `internal/`: db (pgx),
bus (NATS, fail-soft), detect (detection-svc client), opa, risk (+tests), agentrisk (+tests),
compliance (fixture loader +tests), runtime (+tests), httpx. `detection-svc/` is the Python sidecar
(Presidio+YARA+regex; P2 added PHI recognizers: inline MRN, standalone ICD-10, ABA routing).
`policy/opa/` holds `dlp.rego` (+ passing tests). `frontend/` is Next.js with Executive/SOC/
Compliance pages, a BFF under `src/app/api/`, and `src/lib/services.ts` (keep its `TextDecoder`
UTF-8 fix). Compose file `deploy/compose/docker-compose.yml` wires everything; host ports 8081 and
3000 are taken by another project, so discovery is published on **8091** and frontend on **3100**.

### Phase 3 status — CODE WRITTEN, INTEGRATION NOT YET GATED
`agent-svc`, `graph-svc`, `scripts/scenario-5.sh`, and the frontend `/agents` + `/graph` pages
(d3-force animated risk graph) exist and pass `go build ./...`, `go vet ./...`, `go test ./...` and
`next build`. They were written against `docs/PHASE3-INTEGRATION.md` but the live integration +
Gate 3 run did not complete (Docker Desktop crashed mid-run; it has since been recovered). **Your
first job is to finish Phase 3 integration and pass Gate 3**, not to rewrite these services.

### Integration contracts already authored (READ THESE — they are binding)
`docs/PHASE1-INTEGRATION.md`, `PHASE2-INTEGRATION.md`, `PHASE3-INTEGRATION.md`,
`PHASE4-INTEGRATION.md`, `PHASE5-INTEGRATION.md`. `kind v0.23.0` is installed
(`$(go env GOPATH)/bin/kind`) for Phase 5.

---

## 2. THE METHOD THAT HAS BEEN WORKING (keep using it)

Per phase, in this order:
1. **You lock the integration contract first** (a `docs/PHASEn-INTEGRATION.md`): exact ports,
   container names, env vars, request/response shapes, formulas, and the Gate definition. This is
   what lets parallel work converge.
2. **Fan out multiple sub-agents in parallel — one per component/directory** (e.g. Go backend,
   Python, frontend, policy). Each agent builds against the contract and **self-verifies by
   actually running its component** (build, vet, unit tests, image build, a local smoke test) and
   pastes the evidence back. Give each agent tight directory boundaries so they don't collide.
3. **You keep integration and the Verification Gate for yourself — never delegate them.** You wire
   compose, bring the stack up, run the scenario end-to-end, capture real output + dashboard
   screenshots, and write the Gate section of `VERIFICATION.md`.
4. **Operational guardrails**: Docker Desktop on this machine crashes occasionally — if `docker ps`
   fails, kill all `*docker*` processes, relaunch `Docker Desktop.exe`, and poll until `docker ps`
   succeeds before proceeding. Other projects hold host ports 8081/3000 — never touch their
   containers; remap only your own host ports. Use the scratchpad dir for temp files.
5. Mark a chapter per phase. Save durable working-style facts to memory only (not repo trivia).

---

## 3. WHAT "PRODUCTION GRADE" ADDS (on top of the demo MVP)

The original brief targeted a *demo*. "Production grade" means each remaining phase also satisfies
the following, and you add a cross-cutting **Phase 7 — Production Hardening** before final sign-off.
Apply these to new code as you write it; retrofit existing services opportunistically, not in a big
bang.

### 3.1 Reliability & correctness
- Health split: `/healthz` (liveness) **and** `/readyz` (readiness — checks DB/NATS/Neo4j/detection
  dependencies) on every Go service; wire them as compose/K8s probes.
- Graceful shutdown (SIGTERM → drain HTTP server + NATS + pools with a timeout) on every service.
- Context-aware timeouts and bounded retries with backoff on every cross-service call
  (detection-svc, OPA, Neo4j, NATS). No unbounded waits.
- Input validation at every API boundary (reject oversize payloads; cap `/scan` content size with a
  clear 413). Consistent typed error envelope (already `{error,detail}`) everywhere.
- Idempotency where events can be redelivered (NATS/JetStream): dedupe by `eventId`.
- DB migrations: convert `db/postgres/*.sql` into an ordered, versioned migration set run by a
  migrate step (e.g. golang-migrate) rather than only `docker-entrypoint-initdb.d`.

### 3.2 Security (it's a security product — treat its own surface as in-scope)
- **AuthN/AuthZ real on at least the Helm path**: stand up Keycloak (already declared in the OpenAPI
  `oidc` scheme), protect service and dashboard routes with OIDC/JWT validation middleware, and RBAC
  roles (viewer / analyst / admin). On the compose demo path a documented dev bypass is acceptable
  but must be explicit.
- Secrets: nothing hardcoded beyond dev defaults; K8s Secrets on the Helm path; document rotation.
- Container hardening: non-root, read-only rootfs where possible, dropped capabilities, distroless/
  slim base, pinned digests; `.dockerignore` hygiene; Trivy (or grype) image scan in CI with a
  documented severity gate.
- Rate limiting / basic abuse protection on public-facing endpoints; security headers + CORS policy
  on the frontend/BFF.
- Dependency scanning (govulncheck for Go, pip-audit for Python, npm audit for frontend) in CI.

### 3.3 Observability
- OpenTelemetry traces + Prometheus `/metrics` on every service; propagate trace context across the
  pipeline (discovery→prompt→detection→opa→compliance). Ship a Grafana dashboard JSON and a Prom
  scrape config in `deploy/`. Structured JSON logs with request IDs.

### 3.4 Quality gates & CI/CD
- A CI pipeline (GitHub Actions workflows in `.github/workflows/`) that on every push runs: Go
  build/vet/test/govulncheck, `opa test`, Python lint+tests, `next build`+lint, Redocly lint,
  Docker builds, Trivy scan, and `helm lint`. Red CI = not done.
- Test depth: unit tests for all business logic (risk, agentrisk, compliance mapping, OPA policy,
  detection category mapping, copilot grounding); **integration tests** that run the compose stack
  and execute all 6+1 scenarios and assert decisions/violations; a smoke test for each dashboard
  route. Target meaningful coverage on the core packages, not a vanity number.
- Load sanity: a small k6/vegeta script hitting `/prompts/inspect` to show the pipeline holds under
  concurrent inspections; record p50/p95.

### 3.5 Docs & operability
- `docs/DEMO.md` runbook (5-min and 15-min scripts, exact click paths, expected output per scenario).
- `docs/ARCHITECTURE.md` with the compose-path and K8s-path diagrams drawn from the real contracts,
  plus the explicit compose-vs-helm capability matrix.
- `docs/THREAT-MODEL.md` (STRIDE on the platform's own attack surface) and `docs/SECURITY-REVIEW.md`.
- `docs/RUNBOOK.md` (ops: start/stop/reset, common failures + fixes, backup/restore of Postgres/Neo4j).
- `make demo-reset` / `make demo-seed` (or PowerShell + Bash equivalents) for one-command repeatable
  demos between talks.
- Top-level `README.md` kept current with both run paths and the capability matrix.

---

## 4. REMAINING PHASES (each ends at its Verification Gate)

### Phase 3 — Agent Security + Risk Graph  *(finish integration; code exists)*
Follow `docs/PHASE3-INTEGRATION.md`. Recover Docker if needed, `docker compose build` the new
services + rebuild changed ones, `docker compose up -d`, confirm health.
- Run `scripts/scenario-5.sh`: agent `finops-crew-01` (CrewAI) touches Jira→GitHub→Salesforce, does
  a sensitive data_access, then privilege_escalation. Assert the score trajectory **12→24→36→51→76**,
  threshold 70 crossed at the last event, and **exactly one** `violations` row with `agent_id` set
  and a **fixture-sourced** control (OTHER→NIST AI RMF; never hardcoded). Re-run proves idempotency
  (still 76, no second violation).
- graph-svc backfills Neo4j from Postgres + subscribes NATS. Run the demo query
  `phi-to-unauthorized-ai` and return **real rows** (nurse.joy@example.com → gemini →
  patient_records.csv, from Scenario 3). Paste the Cypher + result.
- Screenshot the `/agents` page (risk gauges, threshold-breach state) and the `/graph` animated
  force-directed risk graph (with the executed Cypher shown).
> **Gate 3:** Scenario 5 trajectory + single fixture-sourced violation + idempotency; PHI query real
> rows with pasted Cypher; both screenshots. Record in `VERIFICATION.md`.

### Phase 4 — AI Security Copilot (offline RAG)
Follow `docs/PHASE4-INTEGRATION.md`. Build `copilot-svc` (Go, 8086) + the `/copilot` UI page.
**Deterministic OFFLINE mode is the default** (no network, no keys). For a violation it retrieves the
real evidence bundle (violation row + linked prompt + findings + fixture control [+ agent]) and
composes three grounded views — **executive / technical / compliance** — that contain ONLY retrieved
facts. Live LLM (Claude Messages API) only behind `COPILOT_LIVE=true` + `COPILOT_API_KEY`.
> **Gate 4:** With no network/keys, `GET /copilot/explain/{scenario-1 violationId}` returns all three
> views, each citing the real violation row + fixture control; `/healthz` reports mode `offline`; UI
> screenshot of the three tabs. Record it.

### Phase 5 — Kubernetes Hardening Path (real runtime security)
Follow `docs/PHASE5-INTEGRATION.md`. `kind` cluster `ai-sentinel`; Helm umbrella chart
`deploy/helm/ai-sentinel/` (all services, ConfigMaps for fixtures + OPA policy, Secrets,
ServiceAccounts, NetworkPolicies default-deny, Pod Security Standards restricted). Real **Falco**
(modern_ebpf) + **Tetragon** + **Kyverno** (enforce). Wire falcosidekick → telemetry-generator
`RUNTIME_MODE=ingest` so real Falco events (with `simulated:false`) replace the synthetic feed in the
SOC dashboard, badged **LIVE (Falco)**. If eBPF won't load on this host, record the exact error
honestly and still demonstrate Kyverno fully — never fake runtime events.
> **Gate 5:** `kubectl get pods -n ai-sentinel` all Ready; Kyverno rejects `deploy/k8s-tests/bad-pod.yaml`
> (paste the rejection); a real Falco alert fires on `kubectl exec` shell (paste it), and its arrival
> in the SOC feed if sidekick wiring succeeds; `helm lint` clean. Record it.

### Phase 6 — Conference Hardening
`make demo-reset`/`demo-seed`; `docs/DEMO.md` (5-/15-min scripts); `docs/THREAT-MODEL.md` (STRIDE);
`docs/SECURITY-REVIEW.md`; `docs/ARCHITECTURE.md` (both paths + capability matrix); README refresh.
> **Gate 6:** Fresh clone → `docker compose up` → run the 5-minute script end to end with zero manual
> fixups. Record the full run.

### Phase 7 — Production Hardening  *(new; the "production grade" cross-cut)*
Deliver §3 across the platform: `/readyz` + graceful shutdown + timeouts/retries on every service;
versioned DB migrations; OTel+Prometheus+Grafana; Keycloak-enforced OIDC/RBAC on the Helm path
(documented dev bypass on compose); container hardening + image scanning; CI workflows running the
full quality gate; integration + load tests; `docs/RUNBOOK.md`.
> **Gate 7:** CI green on a clean checkout (all lints/tests/builds/scans); integration test runs all
> scenarios and asserts outcomes; one service demonstrably traces end-to-end through Grafana; an
> unauthenticated request to a protected route is rejected on the Helm path; graceful-shutdown and
> readiness probes observable. Record every item with pasted evidence.

---

## 5. Definition of Done (production grade)
- All Gates 3–7 passed with pasted evidence in `VERIFICATION.md` (0–2 already pass).
- `docker compose up` from a cold clone runs all 6 scenarios with real detection; `/agents`,
  `/graph`, `/copilot` all functional; Copilot works fully offline.
- `helm install ai-sentinel` stands the stack up on `kind` with **real** Falco/Tetragon/Kyverno,
  OIDC/RBAC enforced, NetworkPolicies + PSS restricted.
- CI is green on a clean checkout; integration + load tests pass; images scanned.
- No fabricated control IDs (regional frameworks still carry `TODO-VERIFY` until the product owner
  supplies them). Every simulated feed labelled `SIMULATED` (compose) / real feeds `simulated:false`
  (Helm).
- Deliverables present: source, Dockerfiles, Helm charts + K8s manifests, migrations, OpenAPI specs,
  event schemas, demo datasets + scripts, CI workflows, observability configs, README, architecture
  diagrams, demo runbook, ops runbook, threat model, security review.

## 6. Forbidden
- Reporting any component "done" without pasted run evidence.
- Scripted if/else masquerading as detection in the core pipeline.
- Hallucinated control IDs or clause numbers for any framework.
- Claiming live browser/DNS/proxy/endpoint integrations, or live kernel telemetry on the compose path.
- Starting a phase before the prior gate passes.
- Rewriting already-gated, working code instead of finishing the current phase.
