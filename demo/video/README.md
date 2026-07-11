# AI Sentinel — demo & recording guide

A ~3.5-minute narrated walkthrough of AI Sentinel's control loop:
**discover shadow AI → detect with real DLP → govern → score agent risk →
explain offline**, with honesty built in.

Two ways to produce the video: an **AI-narrated visual player** (zero setup) and
a **real live-terminal** recording (genuine service output on camera).

---

## Option A — the AI-narrated visual player (fastest)

`demo-player.html` is a self-contained, animated CLI demo with AI voice-over.
Open it in a browser and it types the commands, streams the outputs, shows a
synced caption per beat, narrates each step aloud, and opens/closes on branded
title cards. Nothing to install, no stack to bring up.

```
Open demo/video/demo-player.html in Microsoft Edge  (best neural voices),
click "Play with voice-over".
```

To make the video: press **F11** (fullscreen), start your recorder with
**desktop audio** enabled, hit Play. Controls: **Space** pause/resume,
**R** restart, **Esc** menu. Pick a "Microsoft … Online (Natural)" voice.

**The five beats:**

| Beat | Command | The point on screen |
|------|---------|---------------------|
| 1 · Detect  | `scenario-1.sh` → `/scan`, OPA `/decision` | real Presidio finds 113 items (PII×100, PCI×10) → OPA **BLOCK** |
| 2 · Govern  | `/prompts/inspect` | risk **100/100**, payload **redacted in place**, mapped to GDPR/ISO/NIST (regional = `TODO-VERIFY`) |
| 3 · Agents  | `scenario-5.sh`, `/graph/query/…` | agent risk **12→76**, crosses 70 → **one** violation; Neo4j PHI-exposure path |
| 4 · Copilot | `/copilot/ask` | offline, grounded answer + citations; refuses to hallucinate the weather |
| 5 · Honesty | `/copilot/ask` posture | real per-framework counts; **SIMULATED** telemetry + `TODO-VERIFY` controls, never faked |

Beat 3 is the standout — most tools stop at prompts; AI Sentinel scores
autonomous **agents** and renders the agent→tool→data attack path in a graph.

**Narration assets** (single source of truth = the player):

- `narration.txt` — the voice-over script (the player reads these same lines).
- `narration.mp3` (~2.5 MB) / `narration.wav` (~7 MB) — TTS render for a scratch
  track, a VO artist, or muxing under a silent capture. Embed the MP3.

Regenerate the audio after editing the script (Windows PowerShell):

```powershell
Add-Type -AssemblyName System.Speech
$s = New-Object System.Speech.Synthesis.SpeechSynthesizer
$s.SetOutputToWaveFile("demo/video/narration.wav")
$b = New-Object System.Speech.Synthesis.PromptBuilder
Get-Content demo/video/narration.txt | % { $b.AppendText($_); $b.AppendBreak("Large") }
$s.Speak($b); $s.Dispose()
```

## Option B — real live terminal (for the Briefing itself)

Unlike a single-binary tool, AI Sentinel is a multi-service stack, so a live
recording needs the demo path up first:

```sh
cd deploy/compose && cp .env.example .env && docker compose up -d
docker compose ps          # wait until every service is healthy
```

Then drive the same beats live (ports per docker-compose; dashboards at
http://localhost:3100):

```sh
./scripts/scenario-1.sh                                   # Beat 1: detect + OPA BLOCK
curl -s localhost:8082/prompts/inspect -d @prompt.json    # Beat 2: score + redact + map
./scripts/scenario-5.sh                                   # Beat 3: agent risk 12→76, one violation
curl -s localhost:8086/copilot/ask -d '{"question":"Which agent is the riskiest?"}'   # Beat 4
curl -s localhost:8086/copilot/ask -d '{"question":"What is our compliance posture?"}' # Beat 5
```

Record at 1080p with a large terminal font; narrate over it using `narration.txt`.

## IMPORTANT — provenance of the on-screen output

Every line in the visual player is **real output captured from the Gate 1–5
verification runs** recorded in [`../../VERIFICATION.md`](../../VERIFICATION.md)
— the 113-finding scan, the OPA BLOCK, the `12→24→36→51→76` agent trajectory,
the Neo4j PHI-exposure rows, and the copilot's grounded answers are all verbatim
from those runs. They were **not** re-captured for this player (the recorded
compliance still shows Gate 6, the cold-clone `docker compose up` run, as
pending). Before you present, do a live pass with Option B so your on-camera
numbers come from a fresh run of your own stack.

## Recording tips

- **macOS:** QuickTime / ⌘⇧5. **Linux:** OBS. **Windows:** OBS or Game Bar (`Win+G`).
- Upload unlisted to YouTube; that link is your Black Hat CFP video sample.

## A note on honesty (it's the product's whole thesis)

AI Sentinel's design choice — and this demo's — is that **detection is real**
(Presidio + YARA + OPA), **runtime telemetry is synthetic and labelled
`SIMULATED`** (Brief R4), and **compliance mappings are indicative, with
unverified regional controls left as `TODO-VERIFY`** rather than invented
(Brief R5). The demo leans into that honesty instead of hiding it — which is
exactly what a security audience respects.
