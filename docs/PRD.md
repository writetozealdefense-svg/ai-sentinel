# Product Requirements Document — ZealDefense AI Sentinel™

| | |
|---|---|
| **Product** | ZealDefense AI Sentinel™ — *"Discover. Govern. Secure. Every AI Interaction."* |
| **Category** | AI Security Posture Management (AI-SPM) · Shadow-AI discovery · AI-aware DLP · Agentic-AI security · AI compliance |
| **Version** | PRD v1.0 — 2026-08-19 |
| **Status** | MVP built & verified (Gates 0–5 passed, Gate 6 in progress); business validation 12/12 |
| **Owner** | ZealDefense Product |
| **Audience** | Leadership, Engineering, Sales/Pre-sales, Design partners |

---

## 1. Executive summary

Employees and autonomous agents are already sending corporate data to AI systems — ChatGPT, Claude,
Gemini, Copilot, DeepSeek, Grok, self-hosted models, MCP tools — with no visibility, no control,
and no audit trail. Existing DLP was built for email and file shares, not prompts; existing CASB
sees the domain, not the data; and neither speaks the language of the AI-specific regulations now
arriving (NIST AI RMF, SDAIA GenAI guidelines, EU AI Act) or the regional privacy laws that treat a
pasted prompt as a cross-border disclosure (Saudi PDPL Art. 29).

**AI Sentinel** is a platform that makes enterprise AI usage **visible** (shadow-AI inventory),
**safe** (real-time detection of PII/PHI/PCI/secrets/code in prompts with graduated
ALLOW→WARN→REDACT→BLOCK enforcement), **governed** (every incident mapped to the exact control IDs of
nine compliance frameworks, including Saudi PDPL / NCA ECC / SAMA CSF / SDAIA), and **explainable**
(an offline AI Security Copilot that narrates any incident for the board, the engineer, and the
auditor — with citations, and without sending data to another cloud AI). It also extends the same
model to **agentic AI**: discovering agent frameworks, scoring agent risk 0–100, and alerting once
— not fifty times — when an agent crosses a threshold, with a Neo4j risk graph for instant
incident forensics.

The MVP is built, runs from a cold `docker compose up`, and has passed **six verification gates
with pasted run evidence** plus a **12/12 business-validation suite on fresh, never-seen inputs**.
A Kubernetes hardening path (Helm + real Falco/Tetragon/Kyverno) is also proven. The immediate
go-to-market moment is **Black Hat Europe / GCC CISO events**, where the Saudi-regulation depth is
the differentiator.

---

## 2. Problem statement

### 2.1 The market problem
1. **Shadow AI is ubiquitous and invisible.** Most organizations cannot list which AI services their
   staff use, let alone which data went into them.
2. **Legacy DLP does not see prompts.** It pattern-matches attachments and email; it does not
   classify free-text pasted into a browser tab or an IDE plugin, does not handle JSON/log/code
   formats, and is binary (allow/block) — so employees disable it or route around it.
3. **Agentic AI is a new, ungoverned attack surface.** Autonomous agents (LangGraph, CrewAI,
   AutoGen, OpenAI Agents SDK, MCP) call tools, touch data stores, and escalate privileges with no
   per-agent risk model and no alert discipline.
4. **Compliance teams cannot translate AI incidents into control language.** "A prompt was blocked"
   is not an audit artifact. Boards and regulators ask: which regulation, which article, what
   exposure, what was done.
5. **Regional regulation is sharper than generic tooling.** Saudi PDPL treats disclosure to an
   external AI as a cross-border transfer; NCA ECC-2:2024 and SAMA CSF mandate classification-based
   data protection and DLP; SDAIA's GenAI guidelines name prohibited inputs. Global tools ship
   "GDPR-ish" mappings and nothing for the GCC.

### 2.2 Who feels it
| Persona | Pain | What they need |
|---|---|---|
| **CISO / Head of Security** | No inventory of AI usage; can't answer "are we leaking data to AI?" | A live shadow-AI inventory and proof that sensitive data is stopped before it leaves |
| **SOC analyst** | Alert-blind to AI channels; agent activity looks like noise | Live prompt monitoring, DLP alerts, runtime events, agent-risk alerts with dedup |
| **Compliance / DPO / GRC** | Cannot map AI incidents to controls; regional frameworks uncovered | Per-framework posture scores, violation → control-ID mapping, review-gated evidence |
| **CIO / Board** | Want AI productivity *and* risk control; blunt blocking kills adoption | Graduated enforcement and an executive narrative of posture and incidents |
| **Developer / Platform team** | Need to run this in their estate | Contract-first APIs, compose for demo, Helm for hardened K8s |

---

## 3. Goals, non-goals, and guiding principles

### 3.1 Goals (MVP)
- **G1** Discover and inventory AI usage across the org (sanctioned vs shadow) from telemetry.
- **G2** Detect sensitive data in AI-bound content **for real** — PII, PHI, PCI, secrets, source code —
  across text, CSV, JSON and code formats.
- **G3** Enforce graduated decisions (ALLOW / WARN / REDACT / BLOCK) via policy-as-code, with a
  redaction preview so legitimate work continues.
- **G4** Map every violation to verified control IDs across nine frameworks; surface per-framework
  posture; never invent a clause number.
- **G5** Secure agentic AI: discover agents, score risk 0–100, threshold-alert exactly once, and
  chain exposure in a risk graph for forensics.
- **G6** Explain incidents to three audiences (executive / technical / compliance), grounded in the
  platform's own records, **fully offline**.
- **G7** Run in two modes: a self-contained demo path (compose) and a hardened Kubernetes path with
  real runtime security (Helm + Falco/Tetragon/Kyverno).
- **G8** Demonstrable authenticity: every capability shown is executed live; every synthetic feed is
  labelled; every claim has run evidence.

### 3.2 Non-goals (this release)
- Live integrations with real browser extensions, DNS resolvers, proxies, or endpoint agents
  (telemetry is synthetic and labelled `SIMULATED`; connectors are roadmap).
- Acting as the AI gateway / reverse proxy in the request path of production LLM traffic.
- Model-security testing (prompt-injection red-teaming, jailbreak evaluation, model supply chain).
- Multi-tenant SaaS, billing, or self-service onboarding.
- Replacing a SIEM (we emit to one; we don't become one).

### 3.3 Guiding principles (non-negotiable)
1. **Show, don't assert** — done means run, with pasted output.
2. **Real detection on the demo path** — never scripted if/else.
3. **No invented facts** — compliance control IDs come only from reviewed fixtures; placeholders are
   marked `TODO-VERIFY`; unverified mappings carry a visible REVIEW marker.
4. **Honest labelling** — simulated feeds are labelled `SIMULATED` in code, data, and UI.
5. **Graduated, not blunt** — security that employees do not route around.

---

## 4. Users & use cases

### 4.1 Primary use cases (all implemented and executed)
| # | Use case | Actor | Outcome |
|---|---|---|---|
| UC-1 | Employee uploads a customer CSV to ChatGPT | Finance analyst | Real PII/PCI detection → risk 100 → **BLOCK** → GDPR/ISO/NIST/PDPL/NCA/SAMA/DPDPA violations → live on Executive dashboard |
| UC-2 | Developer pastes source with embedded AWS keys into Claude | Contractor | CODE + SECRET detected → **BLOCK** |
| UC-3 | Nurse pastes patient records into Gemini | Clinician | PHI (MRN, ICD-10, clinical terms) → **BLOCK** → **HIPAA 164.312(e)(1)** + PDPL Art. 35 |
| UC-4 | HR uploads payroll (IBANs, routing numbers) to Copilot | HR admin | PCI → **BLOCK** |
| UC-5 | Autonomous agent touches Jira → GitHub → Salesforce, reads sensitive data, escalates privilege | CrewAI agent | Agent risk 12→24→36→51→**76**; crosses 70; **exactly one** NIST AI RMF violation; repeat events do not re-alert |
| UC-6 | Analyst uses DeepSeek | Any employee | Shadow-AI discovery event → DeepSeek appears in inventory as unsanctioned; benign prompt → **REDACT/52** (graduated) |
| UC-7 | Riyadh bank employee pastes Saudi national IDs, iqama numbers, SA IBANs into ChatGPT | Relationship manager | SAUDI_NATIONAL_ID / SAUDI_MOBILE / IBAN detected → **BLOCK** → **PDPL Art. 29, NCA ECC 2-7-2, SAMA CSF 3.3.8, SDAIA GAIG-GOV-ProhibInput** |
| UC-8 | "Who exposed PHI to unauthorized AI systems?" during an incident | SOC / DPO | Named graph query returns real user → AI → dataset rows in < 300 ms, Cypher shown verbatim |
| UC-9 | "Why was this prompt blocked?" for the board and the auditor | Exec / GRC | Copilot returns executive / technical / compliance views with citations — offline |

### 4.2 Business-validation evidence (12/12 PASS, fresh inputs, 2026-08-18)
Beyond the canned scenarios, a separate suite proves generalization: bulk-PII export blocked (risk
90); a single email → surgical REDACT; a clean prompt → ALLOW in 124 ms; PHI detected in free-text
*prose*; PII hidden inside JSON logs still blocked (format evasion fails); brand-new shadow AI
(Grok) inventoried in 17 ms; agent threshold → one alert despite repeat events; forensic graph
query 264 ms; offline Copilot narrative with 9 citations; Saudi PII → PDPL record; six concurrent
users → all correct decisions in 1.6 s. See `docs/BUSINESS-VALIDATION.md`.

---

## 5. Product scope & functional requirements

Requirements are tagged **[MVP ✅]** (built, gated) or **[ROADMAP]**.

### 5.1 Shadow-AI Discovery (discovery-svc)
- **FR-D1 [MVP ✅]** Ingest discovery events (provider, user, source ∈ browser-ext/DNS/proxy/endpoint,
  action ∈ visit/upload/prompt/download, artifact, bytes). Every event carries `simulated: true` on
  the compose path.
- **FR-D2 [MVP ✅]** Maintain an AI inventory aggregated by provider: usage count, unique users,
  sanctioned flag, first/last seen. Providers: chatgpt, claude, gemini, copilot, deepseek, grok,
  perplexity, ollama, openwebui, mcp, browser-ext.
- **FR-D3 [MVP ✅]** Surface unsanctioned providers as **SHADOW AI** on the Executive dashboard.
- **FR-D4 [ROADMAP]** Real connectors: browser extension, DNS/proxy log ingestion, EDR/endpoint
  agent, SaaS audit-log pull; sanctioned-list management UI.

### 5.2 Prompt Inspection & DLP (prompt-svc + detection-svc)
- **FR-P1 [MVP ✅]** `POST /prompts/inspect` orchestrates: record discovery → detect → risk-score →
  policy decision → persist prompt + findings → publish events → raise violations → return result
  with redacted preview and policy trace.
- **FR-P2 [MVP ✅]** **Detection is real** (Python sidecar): Microsoft Presidio (28 recognizers incl.
  custom PHI: MRN, ICD-10, clinical terms; Saudi national ID/iqama, Saudi mobile), YARA (12 rules:
  AWS/GitHub/API keys, private-key blocks, Bearer/JWT, source-code signatures for Python/JS/C/Java/
  SQL), and custom regex (11 patterns: ABA routing, ZIP, ISO dates, IPv6, etc.). Works on arbitrary
  input; content types text / csv / code / json.
- **FR-P3 [MVP ✅]** Findings carry type, category ∈ {PII, PHI, PCI, SECRET, CODE, OTHER}, engine,
  offsets, confidence. Redaction preview replaces each span with `<TYPE> [REDACTED]`.
- **FR-P4 [MVP ✅]** Deterministic **risk score 0–100**: severity weight (SECRET 40 / PHI 35 / PCI 35 /
  PII 25 / CODE 20) + volume (cap 40) + confidence (cap 15) + unsanctioned provider (+10), clamped.
- **FR-P5 [MVP ✅]** **Policy-as-code (OPA/Rego)** decision with reasons: BLOCK if SECRET/PHI/PCI or
  PII ≥ 5; REDACT if PII < 5; WARN if CODE; ALLOW otherwise. Precedence BLOCK > REDACT > WARN > ALLOW.
- **FR-P6 [MVP ✅]** `GET /prompts` newest-first summaries with derived categories (SOC live feed).
- **FR-P7 [ROADMAP]** Inline enforcement at a gateway/proxy; user coaching pop-up; custom
  recognizer/policy management UI; ML classifiers for document types (contracts, financials).

### 5.3 Compliance Mapping (compliance-svc + fixtures)
- **FR-C1 [MVP ✅]** Control mappings are read **exclusively** from `fixtures/compliance/*.yaml`
  (nine frameworks: GDPR, HIPAA, ISO 27001, NIST AI RMF, **PDPL, NCA ECC-2:2024, SAMA CSF v1.0,
  SDAIA AI**, DPDPA). Services never generate a control ID.
- **FR-C2 [MVP ✅]** On REDACT/BLOCK, one violation per matched framework using the
  highest-severity control whose mapped categories intersect the findings; violations record
  framework, controlId, category, severity, user, prompt/agent, timestamp.
- **FR-C3 [MVP ✅]** Per-framework posture: score (100 − 10 × open violations, clamped), control
  count, open violations, `reviewRequired` (true while any control still carries its REVIEW
  marker). Controls endpoint returns the verbatim review marker.
- **FR-C4 [MVP ✅]** Saudi frameworks are populated from official sources and source-cited
  (`fixtures/compliance/SOURCES.md`), with review markers that clear only on product-owner sign-off.
  NCA AICG-1:2026 is intentionally not shipped while in draft.
- **FR-C5 [ROADMAP]** EU AI Act, PCI DSS 4.0, SOC 2, UAE PDPL, Qatar PDPPL; evidence export
  (PDF/CSV) per framework; auditor read-only role.

### 5.4 Agent Security (agent-svc)
- **FR-A1 [MVP ✅]** Ingest agent telemetry (agentId, framework ∈ langgraph/crewai/autogen/
  openai-agents-sdk/mcp, action ∈ tool_call/data_access/privilege_escalation/external_call, tool,
  target, sensitive, simulated=true).
- **FR-A2 [MVP ✅]** Deterministic **Agent Risk Score 0–100** from full history: +12 per distinct tool
  (cap 36), +15 per sensitive data access (cap 30), +25 privilege escalation, +10 external call.
- **FR-A3 [MVP ✅]** **Threshold 70, crossing semantics**: exactly one violation (category OTHER →
  fixture-mapped NIST AI RMF control) when the score first crosses 70; subsequent events above
  threshold do not re-alert (alert-storm prevention).
- **FR-A4 [MVP ✅]** Agents page: framework badge, owner, tool chips, risk gauge (red at ≥ 70 with
  THRESHOLD BREACH), risk factors, SIMULATED tag.
- **FR-A5 [ROADMAP]** Real agent instrumentation (OpenTelemetry/LangSmith/MCP server hooks), tool
  allow-lists, kill-switch / quarantine action, per-agent policy.

### 5.5 Risk Graph (graph-svc + Neo4j)
- **FR-G1 [MVP ✅]** Graph model: User, Prompt, Model, Agent, Tool, Dataset, Violation; relationships
  USED, ACCESSED, GENERATED, EXPOSED. Backfilled from Postgres on start; kept live via NATS.
- **FR-G2 [MVP ✅]** Named, server-side Cypher queries only (no arbitrary Cypher over the wire):
  `phi-to-unauthorized-ai`, `high-risk-agents`, `blocked-prompts-by-user`, `full-graph`. Responses
  return tabular rows **and** a renderable subgraph **and** the executed Cypher.
- **FR-G3 [MVP ✅]** Animated force-directed graph UI (d3-force on canvas), node colours by label,
  glow on violations/high-risk, Cypher shown verbatim, rows table.
- **FR-G4 [ROADMAP]** Path-finding ("blast radius"), time-travel, export to SIEM/graph tools.

### 5.6 AI Security Copilot (copilot-svc)
- **FR-K1 [MVP ✅]** **Offline deterministic mode is the default**: for a violation, retrieve the
  evidence bundle (violation row, linked prompt, aggregated findings, fixture control, agent if any)
  and compose three grounded views — executive / technical / compliance — using only retrieved
  facts, with citations (violation / control / prompt / finding).
- **FR-K2 [MVP ✅]** `/copilot/ask` with intent routing ("why was the last prompt blocked", "riskiest
  user/agent", "compliance posture"); honest fallback listing what it can answer — never hallucinate.
- **FR-K3 [MVP ✅]** Live LLM mode only behind `COPILOT_LIVE=true` + API key; mode reported in
  `/healthz`; UI badge OFFLINE · DETERMINISTIC vs LIVE.
- **FR-K4 [ROADMAP]** Vector retrieval over historical incidents, remediation playbooks, ticket
  creation (Jira/ServiceNow), natural-language policy authoring.

### 5.7 Dashboards (Next.js 15)
- **FR-U1 [MVP ✅]** **Executive**: AI inventory (SHADOW AI badges, SIMULATED tag), latest inspection
  (decision badge, risk gauge, findings by category, redacted preview, policy trace), open
  violations by framework (REVIEW PENDING markers), Run-Scenario control.
- **FR-U2 [MVP ✅]** **SOC**: live prompt monitoring (4 s poll), DLP alerts, runtime events
  (Falco/Tetragon) with a **SIMULATED** badge on compose and **LIVE (Falco)** on the Helm path.
- **FR-U3 [MVP ✅]** **Compliance**: R5 banner, nine framework cards with scores/REVIEW badges,
  controls drill-down with verbatim review markers, violations table.
- **FR-U4 [MVP ✅]** **Agents**, **Risk Graph**, **Copilot** pages as above; shared nav; BFF route
  handlers (server-side, container DNS, never-throw envelopes, explicit UTF-8 decode).
- **FR-U5 [ROADMAP]** Role-aware views, saved filters, dark/light theme toggle, multi-tenant scoping.

### 5.8 Runtime Security & Platform Hardening
- **FR-R1 [MVP ✅]** Compose path: `telemetry-generator` emits SIMULATED Falco/Tetragon-style
  events; every event `simulated:true`; UI labelled.
- **FR-R2 [MVP ✅]** Helm path (`deploy/helm/ai-sentinel`): all services in namespace `ai-sentinel`,
  Pod Security Standards restricted, NetworkPolicies default-deny, per-service ServiceAccounts,
  Secrets; **real** Falco + falcosidekick, Tetragon TracingPolicy, Kyverno enforce policies.
  `telemetry-generator RUNTIME_MODE=ingest` accepts falcosidekick webhooks (`simulated:false`).
  Gate 5: 14/14 pods Running, Kyverno rejects a privileged pod, Falco alert on exec shell (eBPF
  host limits recorded honestly).
- **FR-R3 [ROADMAP]** Keycloak OIDC + RBAC enforced on all routes (OIDC scheme already declared in
  every OpenAPI spec); image signing; SBOM.

---

## 6. Non-functional requirements

| Area | Requirement | MVP status |
|---|---|---|
| **Latency** | Inspect decision p95 < 3 s for ≤ 10 KB payloads after warm-up; clean prompts < 200 ms; graph named query < 1 s | ✅ Observed: 0.1–2.1 s inspections; 264 ms graph; 124 ms clean prompt |
| **Concurrency** | Consistent decisions under ≥ 6 concurrent inspections; no errors | ✅ 6 parallel in 1.6 s, 0 errors |
| **Availability (demo)** | Cold `docker compose up` → all 14 containers healthy; one-command reset/seed/status | ✅ `scripts/demo-up.sh`, `demo-reset.sh`, `demo-seed.sh`, `demo-status.sh` |
| **Offline operation** | Full demo (incl. Copilot) with no internet and no API keys | ✅ Copilot mode `offline` by default |
| **Security of the platform** | Non-root containers, distroless Go images, read-only fixture mounts, PSS restricted + NetworkPolicies on K8s, no arbitrary Cypher/Rego over the wire, typed error envelopes | ✅ (compose: dev defaults; K8s: hardened) · Auth enforcement ROADMAP |
| **Data integrity** | Postgres is source of truth; Neo4j idempotent MERGE backfill; NATS fail-soft; JetStream for violations | ✅ |
| **Auditability** | Every violation carries framework/controlId/severity/category/time and links to prompt or agent; review markers surfaced; SOURCES.md audit trail | ✅ |
| **Honesty & labelling** | `simulated` flag in schemas/data/UI; REVIEW markers; no invented control IDs | ✅ enforced by contract and tests |
| **Observability** | Structured logs; `/healthz` everywhere; OTel traces + Prometheus/Grafana | Logs/health ✅ · OTel/Prom ROADMAP |
| **Portability** | Docker Compose (demo) + Helm on kind/minikube (hardened); Windows/macOS/Linux hosts | ✅ |
| **Accessibility** | Keyboard focus states, legible contrast in dark theme | Partial ✅ |

---

## 7. System architecture (as built)

**Services (Go + Gin unless noted)** — discovery-svc (8081), prompt-svc (8082), agent-svc (8083),
compliance-svc (8084), graph-svc (8085), copilot-svc (8086), telemetry-generator (8087),
**detection-svc (Python FastAPI, 8000)**, OPA (8181), frontend (Next.js 15, 3000).
**Datastores** — PostgreSQL (system of record), Redis, Neo4j (risk graph), NATS + JetStream (bus).
**Contracts first** — OpenAPI 3.1 per service (`contracts/openapi/`, Redocly-lint clean), NATS
subjects + JSON Schemas (`contracts/events/`), Postgres DDL, Neo4j constraints; Go server stubs and
TS client generated from the specs.
**Event flow** — `ai.discovery.observed` → `ai.prompt.inspected` → `ai.violation.raised` →
`ai.agent.activity` → `ai.runtime.observed`; graph-svc and compliance-svc subscribe.
**Two run targets** — compose (demo; synthetic telemetry labelled) and Helm/kind (hardened; real
Falco/Tetragon/Kyverno). Capability matrix in `README.md`.

---

## 8. Success metrics

### 8.1 Product / technical KPIs (measured)
- Verification gates passed with evidence: **6 of 7** (Gate 6 cold-clone run in progress).
- Business-validation cases passed on fresh inputs: **12/12**.
- Detection coverage: PII, PHI, PCI, SECRET, CODE across text/CSV/JSON/code; Saudi identifiers.
- False-positive control: clean prompts ALLOW (0 findings) — tracked per release.
- Decision latency p95 < 3 s; clean prompt < 200 ms; graph query < 1 s.
- Alert discipline: 1 violation per agent threshold crossing (0 duplicates).

### 8.2 Business KPIs (targets for design-partner phase)
- **Time-to-visibility**: shadow-AI inventory populated within 24 h of deployment.
- **Prevented exposure**: # sensitive-data egress events blocked/redacted per month, by category.
- **Adoption friction**: % of inspected prompts ALLOWed without findings (target ≥ 85%).
- **Audit efficiency**: time to produce a framework-mapped incident narrative (target < 1 min).
- **Forensics**: time to answer "who exposed X to which AI" (target < 1 min).
- **Commercial**: 3 GCC design partners (bank / healthcare / government) within two quarters of
  Black Hat Europe; conversion of ≥ 1 to paid pilot.

---

## 9. Release plan & milestones

| Phase | Scope | Gate | Status |
|---|---|---|---|
| 0 | Contracts, schemas, datastores | Gate 0 | ✅ 2026-06-21 |
| 1 | Vertical slice — Scenario 1 end-to-end | Gate 1 | ✅ 2026-06-25 |
| 2 | Scenarios 2/3/4/6 + SOC & Compliance dashboards | Gate 2 | ✅ 2026-06-25 |
| 3 | Agent security + Neo4j risk graph (Scenario 5) | Gate 3 | ✅ 2026-07-03 |
| 4 | AI Security Copilot (offline RAG) | Gate 4 | ✅ 2026-07-03 |
| 5 | Kubernetes hardening path (Helm, Falco/Tetragon/Kyverno) | Gate 5 | ✅ 2026-07-08 |
| 5b | GCC depth: Saudi detectors, PDPL/NCA/SAMA/SDAIA fixtures, Scenario 7 | — | ✅ 2026-08-10 |
| 6 | Conference hardening: one-command up/reset/seed/status, DEMO.md, business validation, logo | Gate 6 | 🔶 in progress (cold-clone run pending) |
| 7 | Production hardening: OIDC/RBAC, OTel/Prometheus/Grafana, CI quality gates, migrations, load tests, threat model & security review | Gate 7 | ROADMAP |
| 8 | Connectors & enforcement: real telemetry connectors, inline gateway enforcement, agent instrumentation, EU AI Act & more frameworks | — | ROADMAP |

---

## 10. Risks & mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Audience assumes telemetry is real | Credibility | `SIMULATED` labels everywhere; say it out loud in the demo; connectors on roadmap |
| A regional control ID is challenged by someone who wrote it | Credibility / legal | Fixtures source-cited (SOURCES.md), REVIEW markers visible until product-owner sign-off; `TODO-VERIFY` where unverified; never generated |
| Detection false positives/negatives | Adoption / trust | Real engines + negative-control tests in the validation suite; tunable recognizers; graduated enforcement |
| Conference network/hardware fragility | Demo failure | Offline Copilot; one-command reset; pre-seeded data; status check; recovery recipe; rehearsal recording |
| Docker Desktop instability on the demo laptop | Demo failure | Documented recovery; 3-min warm-up before stage; IPv4 scripts; backup machine |
| No auth on compose path | Security optics | Explicit dev-bypass statement; OIDC declared in contracts; enforced on Helm path in Phase 7 |
| Scope creep toward "AI gateway" | Focus | Non-goals explicit; gateway enforcement is Phase 8 |

---

## 11. Open questions
1. Which Saudi fixtures should have REVIEW markers cleared first (product-owner sign-off order)?
2. First design-partner vertical: bank (SAMA) vs healthcare (PHI/PDPL Art. 35) vs government (NCA)?
3. Telemetry connector priority: browser extension vs proxy/DNS log ingestion vs EDR?
4. Should the Copilot's live mode default to a regional/sovereign LLM endpoint for GCC customers?
5. Packaging: single-tenant on-prem/Helm only for v1, or a hosted option?

---

## 12. Appendix — where the evidence lives
- `VERIFICATION.md` — gate-by-gate pasted run evidence (Gates 0–5).
- `docs/BUSINESS-VALIDATION.md` + `scripts/business-validation.py` — 12 executable business cases.
- `docs/DEMO.md` — 5-minute and 15-minute conference scripts; `scripts/demo-*.sh`.
- `docs/PHASE{1..5}-INTEGRATION.md` — binding integration contracts per phase.
- `contracts/openapi/*.yaml`, `contracts/events/*.schema.json` — API and event contracts.
- `fixtures/compliance/*.yaml` + `SOURCES.md` — control mappings and their sources.
- `deploy/compose/docker-compose.yml`, `deploy/helm/ai-sentinel/` — the two run paths.
