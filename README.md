# ZealDefense AI Sentinel™

> *Discover. Govern. Secure. Every AI Interaction.*

Enterprise Shadow-AI discovery, governance, DLP, agent-security and compliance platform —
conference-demo MVP. This repo is built **phase by phase**, each gated by recorded run
evidence in [`VERIFICATION.md`](VERIFICATION.md). Nothing is "done" without pasted output.

## Two run targets (Brief R1)

| Path | Command | Purpose | Runtime security |
|------|---------|---------|------------------|
| **Demo path** | `cd deploy/compose && cp .env.example .env && docker compose up` | The conference demo. Real detection, synthetic (`SIMULATED`) telemetry. | Falco/Tetragon/Kyverno **NOT** present — they need a real kernel. SOC runtime widgets are fed by a `SIMULATED` generator, clearly labelled. |
| **Hardening path** | `helm install ai-sentinel ...` (Phase 5) | The architecture story on `kind`/`minikube`. | **Real** Falco, Tetragon, Kyverno, NetworkPolicies, RBAC, PSS. |

## Reality-gap decisions (implemented as specified)
- **R2** — Detection is a Python `detection-svc` (Presidio + YARA + regex), consumed by Go services. Detection is real, never scripted.
- **R3** — Copilot defaults to deterministic **offline** mode; live LLM gated behind `COPILOT_LIVE=true`.
- **R4** — All telemetry is synthetic and labelled `SIMULATED` in code, data, and UI.
- **R5** — Compliance control IDs come only from `fixtures/compliance/*.yaml`; regional frameworks (NCA ECC / SAMA CSF / PDPL / DPDPA) are placeholders awaiting product-owner verification.
- **R5 (Singapore)** — Singapore control IDs likewise come only from `fixtures/compliance/*.yaml`, across nine source-cited packs (PDPA / MAS TRM / MAS AIRG / PDPC GenAI / PDPC AI-RDS / CSA Securing AI / CSA Agentic AI / CSA Cyber Essentials / IMDA Agentic MGF). Every control carries a review marker until the product owner verifies it against its cited source (audit trail: [`SOURCES-SG.md`](fixtures/compliance/SOURCES-SG.md)); MAS AIRG IDs stay `PROPOSED-*` placeholders until mapped to the final Guidelines issued 7 Oct 2026.

## Repo layout
```
contracts/       OpenAPI 3.1 specs (source of truth), NATS event schemas, generated stubs
fixtures/        compliance mappings (Brief R5) + demo datasets
deploy/compose   demo path (docker-compose)
deploy/helm      hardening path (Phase 5)
db/              PostgreSQL DDL + Neo4j graph model
services/        Go (Gin) microservices + telemetry-generator
detection-svc/   Python FastAPI detection sidecar (Brief R2)
frontend/        Next.js 15 dashboards (Executive / SOC / Compliance)
policy/opa       Open Policy Agent decision policies
docs/            demo runbook, threat model, architecture diagrams
```

## Demo

See [`docs/DEMO.md`](docs/DEMO.md) for the timed 5- and 15-minute demo scripts
(cold clone → `docker compose up` → six attack scenarios with real detection).

## Build status
| Phase | Gate | Status |
|-------|------|--------|
| 0 — Contracts & datastores | Gate 0 | ✅ **PASS** (2026-06-21) |
| 1 — Vertical slice (Scenario 1) | Gate 1 | ✅ **PASS** (2026-06-25) |
| 2 — Breadth (Scenarios 2,3,4,6) | Gate 2 | ✅ **PASS** (2026-06-25) |
| 3 — Agent security + risk graph | Gate 3 | ✅ **PASS** (2026-07-03) |
| 4 — AI Security Copilot (offline RAG) | Gate 4 | ✅ **PASS** (2026-07-03) |
| 5 — Kubernetes hardening path | Gate 5 | ✅ **PASS** (2026-07-08, eBPF host limits recorded honestly) |
| 6 — Conference hardening | Gate 6 | 🔶 in progress — demo scripts + repo hygiene landed; full cold-clone run pending |

See [`contracts/gen/README.md`](contracts/gen/README.md) for regenerating client/server stubs.

## License

Apache-2.0 — see [LICENSE](LICENSE).
