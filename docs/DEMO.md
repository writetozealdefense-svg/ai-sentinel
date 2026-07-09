# AI Sentinel — Demo Scripts

Two timed scripts for running the conference demo from a cold clone. Both use
the **compose path** (real detection, synthetic `SIMULATED` telemetry — see
README "Two run targets").

## Prerequisites

- Docker Desktop / Docker Engine with Compose v2
- `bash`, `curl`, `python3` on PATH (the scenario scripts use them)
- ~4 GB free RAM for the stack

## Setup (once, ~3 min on first pull/build)

```bash
git clone https://github.com/writetozealdefense-svg/ai-sentinel
cd ai-sentinel/deploy/compose
cp .env.example .env          # demo defaults; NOT for production
docker compose up -d --build
docker compose ps             # wait until every service reports healthy
```

Dashboards: **http://localhost:3100** (Executive / SOC / Compliance).

## The 5-minute script

| ⏱ | Beat | Action | What the audience sees |
|---|------|--------|------------------------|
| 0:00 | Hook | Open the Executive dashboard | Shadow-AI usage discovered across the org: providers, users, risk posture |
| 0:45 | Scenario 1 — PII exfil | `./scripts/scenario-1.sh` | An employee pastes `customer_database.csv` into ChatGPT; **real** detection (Presidio + YARA + regex) flags PII in the prompt; violation lands in the SOC feed |
| 1:45 | Scenario 2 — source-code leak | `./scripts/scenario-2.sh` | A developer uploads `source_code_leak.py` (embedded AWS keys) to Claude → CODE + SECRET findings, WARN/BLOCK decision |
| 2:30 | Agent security | Open `/agents`, then `./scripts/scenario-5.sh` | Agent `finops-crew-01` risk score climbs 12 → 76 across five telemetry events; crossing 70 emits exactly one violation (NIST AI RMF); the Neo4j risk graph chains agent → tool → data-store exposure |
| 3:30 | Copilot | Open `/copilot`, ask "summarise today's highest-risk AI activity" | Offline deterministic RAG answer citing the actual findings (no external LLM call — `COPILOT_LIVE=false`) |
| 4:15 | Compliance | Open the Compliance dashboard | Violations mapped to framework controls from `fixtures/compliance/*.yaml` |
| 4:45 | Close | Point at the SOC feed's `SIMULATED` badges | Honesty story: detection is real, telemetry is labelled synthetic; the Helm path runs real Falco/Tetragon/Kyverno |

## The 15-minute script

Run the 5-minute script, then add:

1. **Scenario 3 + 4 + 6** (`./scripts/scenario-3.sh`, `-4`, `-6`) — breadth:
   a nurse uploads patient records to Gemini (PHI → BLOCK, HIPAA violation),
   HR uploads a payroll sheet to Copilot (PCI → BLOCK), and an analyst uses
   DeepSeek (shadow-AI discovery lands in the inventory).
2. **Risk graph deep-dive** — in `/graph`, walk the PHI exposure path
   (screenshots in `docs/screenshots/phase3-graph-phi.png` show the expected
   view).
3. **Copilot personas** — re-ask the same question as Executive vs Technical vs
   Compliance persona (`docs/screenshots/phase4-copilot-*.png`).
4. **Hardening path story** — `deploy/helm` on kind/minikube: real Falco alert
   on `kubectl exec` shell, Kyverno rejecting `deploy/k8s-tests/bad-pod.yaml`
   (evidence: VERIFICATION.md Gate 5), NetworkPolicies + restricted PSS.

## Reset between runs

```bash
cd deploy/compose
docker compose down -v        # drops datastore volumes (full reset)
docker compose up -d
```

## Troubleshooting

- **A service is unhealthy** — `docker compose logs <service>`; most first-run
  failures are the datastores still initialising; `docker compose ps` again.
- **Port clash** — frontend is host-mapped to `3100`, prompt-svc to `8082`;
  edit `deploy/compose/docker-compose.yml` port mappings if taken.
- **Scenario script hangs** — the first detection call warms up Presidio
  models; give `scenario-1.sh` up to its 120 s `--max-time` once.
