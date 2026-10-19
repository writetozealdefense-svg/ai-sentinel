# AI Sentinel — Demo Scripts

Two timed scripts for running the conference demo from a cold clone. Both use
the **compose path** (real detection, synthetic `SIMULATED` telemetry — see
README "Two run targets").

## Prerequisites

- Docker Desktop / Docker Engine with Compose v2
- `bash`, `curl`, `python3` on PATH (the scenario scripts use them)
- ~4 GB free RAM for the stack

## Setup (once, ~3–6 min on first pull/build)

```bash
git clone https://github.com/writetozealdefense-svg/ai-sentinel
cd ai-sentinel
bash scripts/demo-up.sh       # build -> up -> wait healthy -> seed ALL scenarios
bash scripts/demo-status.sh   # readiness check: health + seeded counts + URLs
```

`demo-up.sh` leaves every dashboard pre-populated (all seven scenarios run through the real
pipeline). Use `--no-seed` for an empty stack if you prefer to fire scenarios live on stage.
With `make` available, `make demo-up` / `make demo-seed` / `make demo-reset` / `make demo-status`
wrap the same scripts.

Dashboards: **http://localhost:3100** (Executive / SOC / Compliance / Agents / Risk Graph / Copilot).

## The 5-minute script

| ⏱ | Beat | Action | What the audience sees |
|---|------|--------|------------------------|
| 0:00 | Hook | Open the Executive dashboard | Shadow-AI usage discovered across the org: providers, users, risk posture |
| 0:45 | Scenario 1 — PII exfil | `./scripts/scenario-1.sh` | An employee pastes `customer_database.csv` into ChatGPT; **real** detection (Presidio + YARA + regex) flags PII in the prompt; violation lands in the SOC feed |
| 1:45 | Scenario 2 — source-code leak | `./scripts/scenario-2.sh` | A developer uploads `source_code_leak.py` (embedded AWS keys) to Claude → CODE + SECRET findings, WARN/BLOCK decision |
| 2:30 | Agent security | Open `/agents`, then `./scripts/scenario-5.sh` | Agent `finops-crew-01` risk score climbs 12 → 76 across five telemetry events; crossing 70 emits exactly one violation (NIST AI RMF); the Neo4j risk graph chains agent → tool → data-store exposure |
| 3:30 | Copilot | Open `/copilot`, ask "summarise today's highest-risk AI activity" | Offline deterministic RAG answer citing the actual findings (no external LLM call — `COPILOT_LIVE=false`) |
| 4:15 | Compliance — **the GCC moment** | `./scripts/scenario-7.sh`, then open the Compliance dashboard | A Riyadh employee pastes `saudi_customer_data.csv` (Saudi national IDs, iqama numbers, SA IBANs, Saudi mobiles) into ChatGPT → the regex engine flags `SAUDI_NATIONAL_ID`/`SAUDI_MOBILE`; violations map to **real Saudi control IDs**: PDPL Article 29 (cross-border), NCA ECC 2-7-2, SAMA CSF 3.3.8 (DLP), SDAIA GenAI prohibited-inputs |
| 4:45 | Close | Point at the SOC feed's `SIMULATED` badges | Honesty story: detection is real, telemetry is labelled synthetic; every Saudi control ID is source-cited in `fixtures/compliance/SOURCES.md` and review-gated before its badge clears |

## The 15-minute script

Run the 5-minute script, then add:

1. **Scenario 3 + 4 + 6** (`./scripts/scenario-3.sh`, `-4`, `-6`) — breadth:
   a nurse uploads patient records to Gemini (PHI → BLOCK, HIPAA **and** PDPL
   Article 35 — criminal exposure for sensitive-data disclosure), HR uploads a
   payroll sheet to Copilot (PCI → BLOCK), and an analyst uses DeepSeek
   (shadow-AI discovery lands in the inventory).
2. **GCC compliance deep-dive** — on the Compliance dashboard, open each Saudi
   framework's controls view: NCA ECC-2:2024, SAMA CSF v1.0, PDPL (M/19 as
   amended M/148), SDAIA AI. Talk track: every control ID was verified against
   the official source (`fixtures/compliance/SOURCES.md` is the audit trail);
   `reviewRequired` badges clear only on human sign-off — no invented clause
   numbers, ever (Brief R5). Mention SDAIA's live enforcement (48 decisions by
   Jan 2026) and the pending NCA AI Cybersecurity Guidelines (AICG-1:2026).
3. **Risk graph deep-dive** — in `/graph`, walk the PHI exposure path
   (screenshots in `docs/screenshots/phase3-graph-phi.png` show the expected
   view).
4. **Copilot personas** — re-ask the same question as Executive vs Technical vs
   Compliance persona (`docs/screenshots/phase4-copilot-*.png`).
5. **Hardening path story** — `deploy/helm` on kind/minikube: real Falco alert
   on `kubectl exec` shell, Kyverno rejecting `deploy/k8s-tests/bad-pod.yaml`
   (evidence: VERIFICATION.md Gate 5), NetworkPolicies + restricted PSS.

## Reset between runs (between talks, ~2 min)

```bash
bash scripts/demo-reset.sh    # down -v (full data wipe) -> up -> re-seed
```

`--no-seed` resets to an empty stack (fire scenarios live instead). `make demo-reset` is the
same. To just stop without wiping: `make demo-down` (or `cd deploy/compose && docker compose down`).

## Day-of checklist (conference)

30 minutes before the talk, on the presentation machine:

1. `bash scripts/demo-up.sh` (or `demo-reset.sh` if the stack ran earlier) — **needs no
   internet once images are built**; the whole demo is offline-safe (Copilot is
   deterministic-offline by default, `COPILOT_LIVE=false`).
2. `bash scripts/demo-status.sh` — all 8 services OK, seeded counts non-zero, copilot mode
   `offline`.
3. Open the six dashboard tabs in order: `/` → `/soc` → `/compliance` → `/agents` → `/graph`
   → `/copilot`; leave them open (Next.js pages are compiled on first hit — pre-warm them).
4. Keep one terminal at the repo root for the live scenario beats (`./scripts/scenario-1.sh`
   etc. — decide beforehand whether you demo them live or rely on the seed).
5. If Docker Desktop wedges (known on some Windows machines): kill all `*docker*` processes,
   relaunch Docker Desktop, wait for `docker ps`, then `bash scripts/demo-up.sh --no-build`.
6. Screen-record a full rehearsal once (`demo-reset.sh` afterwards) — it doubles as the
   backup video if the venue machine dies.

## Troubleshooting

- **A service is unhealthy** — `docker compose logs <service>`; most first-run
  failures are the datastores still initialising; `docker compose ps` again.
- **Port clash** — frontend is host-mapped to `3100`, prompt-svc to `8082`;
  edit `deploy/compose/docker-compose.yml` port mappings if taken.
- **Scenario script hangs** — the first detection call warms up Presidio
  models; give `scenario-1.sh` up to its 120 s `--max-time` once.
- **Scenario 7 second step errors** — the compliance API host-mapping is
  optional; the violations are always visible on the Compliance dashboard.
