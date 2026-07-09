# Phase 3 Integration Contract (Agent Security + Risk Graph)

Phases 0–2 are DONE and gated (see VERIFICATION.md). The compose stack is running. Phase 3 adds
two Go services (`agent-svc`, `graph-svc`), Scenario 5, the Neo4j risk graph, and two frontend
pages. **Do not deviate from the interfaces below.**

## Services, ports (compose path)
| Component | Container   | Internal URL              | Host port |
|-----------|-------------|---------------------------|-----------|
| agent-svc | `agent-svc` | http://agent-svc:8083     | 8083      |
| graph-svc | `graph-svc` | http://graph-svc:8085     | 8085      |

Env (in addition to Phase 1 vars):
```
NEO4J_URL=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=sentinel_dev_pw
```

## Interface 1 — agent-svc (Go/Gin, port 8083; contract = contracts/openapi/agents.yaml)
- `POST /agents/telemetry` — ingest a SIMULATED agent activity event
  `{agentId, framework, action: tool_call|data_access|privilege_escalation|external_call,
    tool?, target?, sensitive?, owner?, name?, simulated: true, timestamp}`.
  Upserts `agents` row (framework, owner, name, simulated=true, last_seen), inserts
  `agent_activity`, recomputes the risk score, publishes `ai.agent.activity` (schema
  contracts/events/agent.activity.schema.json). Response 202
  `{accepted: true, agentId, riskScore}`.
- `GET /agents` — list agents `{agentId, name, framework, owner, tools[], riskScore,
  riskFactors[], simulated, lastSeen}` (tools = distinct tools from activity).
- `GET /agents/{id}` — one agent + recent activity (extra field `activity: [...]` is fine).

**Agent Risk Score (deterministic, 0–100, computed from that agent's full activity history):**
```
+12 per distinct tool used            (cap 36)
+15 per sensitive data_access event   (cap 30)
+25 if any privilege_escalation event
+10 if any external_call event
score = clamp(sum, 0, 100)
```
**Threshold = 70.** On a telemetry event that moves the score from <70 to >=70 (crossing, not
every event above it): emit ONE `ai.violation.raised` mapped from category **OTHER** via the
existing `internal/compliance` fixtures (NIST AI RMF `MAP 1.1` maps OTHER — fixture-sourced per
R5; do NOT hardcode the control ID), with `agentId` set, severity from the control, description
"Agent risk threshold breached (score N): <riskFactors>". Also insert the `violations` row
(prompt_id NULL, agent_id set). riskFactors = human strings, e.g. "3 tools (jira, github,
salesforce)", "sensitive data access (salesforce.accounts)", "privilege escalation".

## Interface 2 — graph-svc (Go/Gin, port 8085; contract = contracts/openapi/graph.yaml)
Populates and queries the Neo4j risk graph.

**Population (dual mode):**
1. **Backfill on startup** from Postgres (source of truth): users (from prompts.user_email +
   discovery_events), models (provider, sanctioned), prompts (id, decision, riskScore,
   categories from findings, artifactName), datasets (from prompts.artifact_name), violations,
   agents + tools (from agent_activity).
2. **Live**: subscribe NATS `ai.prompt.inspected`, `ai.violation.raised`, `ai.agent.activity`,
   `ai.discovery.observed` and MERGE incrementally. Fail-soft if NATS is down.

**Graph model (db/neo4j/schema.cypher constraints; run them idempotently on startup):**
```
(u:User {email})-[:GENERATED]->(p:Prompt {id, decision, riskScore, categories: [..], scenario})
(u:User)-[:USED]->(m:Model {provider, sanctioned})
(p:Prompt)-[:USED]->(m:Model)
(p:Prompt)-[:EXPOSED]->(d:Dataset {name})
(p:Prompt)-[:EXPOSED]->(v:Violation {id, framework, controlId, severity, category})
(a:Agent {id, framework, riskScore, owner})-[:USED]->(t:Tool {name})
(a:Agent)-[:ACCESSED]->(d:Dataset)
```
MERGE on natural keys (email/provider/id/name) so re-runs are idempotent.

**Named queries** (`GET /graph/queries`, `POST /graph/query/{name}` per graph.yaml; response =
`{rows, nodes, edges, cypher}` where nodes/edges form the renderable subgraph):
1. `phi-to-unauthorized-ai` (THE demo query): users who uploaded PHI to unauthorized AI systems.
   ```cypher
   MATCH (u:User)-[:GENERATED]->(p:Prompt)-[:USED]->(m:Model {sanctioned:false})
   WHERE 'PHI' IN p.categories
   OPTIONAL MATCH (p)-[:EXPOSED]->(d:Dataset)
   RETURN u.email AS user, m.provider AS aiSystem, d.name AS dataset,
          p.riskScore AS riskScore, p.decision AS decision
   ```
2. `high-risk-agents`: agents with riskScore >= 70 and the tools they use.
3. `blocked-prompts-by-user`: users → blocked prompts → models (overview subgraph).
4. `full-graph`: bounded (LIMIT ~150 nodes) whole-graph view for the animated UI.
Return BOTH tabular `rows` AND the `nodes`/`edges` subgraph for each query. Include the exact
`cypher` string in the response (UI shows it for authenticity).

## Interface 3 — Scenario 5 trigger (`scripts/scenario-5.sh`)
POSTs a telemetry sequence for agent `finops-crew-01` (framework `crewai`, owner
`ops.lead@example.com`, name "FinOps Automation Crew") to `http://localhost:8083/agents/telemetry`:
1. `tool_call` jira (target `PROJ-1042`)            → score 12
2. `tool_call` github (target `payments-service`)   → score 24
3. `tool_call` salesforce (target `accounts API`)   → score 36
4. `data_access` salesforce, `target: salesforce.accounts`, `sensitive: true` → score 51
5. `privilege_escalation` (target `github org admin`) → score 76  ← crosses 70, ONE violation emitted
Print each response (accepted + riskScore) then `GET /agents/finops-crew-01`.

## Interface 4 — frontend additions
Nav gains **Agents** (`/agents`) and **Risk Graph** (`/graph`). New BFF routes:
- `GET /api/agents` → `${AGENT_SVC_URL}/agents` (default `http://agent-svc:8083`)
- `GET /api/graph/queries` → `${GRAPH_SVC_URL}/graph/queries` (default `http://graph-svc:8085`)
- `POST /api/graph/query/[name]` → `${GRAPH_SVC_URL}/graph/query/{name}`

**/agents page**: agent cards (name, framework badge, owner, tool chips, animated 0–100 risk
gauge — red when >=70, riskFactors list, SIMULATED tag). Poll ~4s.
**/graph page**: the ANIMATED risk graph — force-directed layout (d3-force allowed) rendered on
canvas/SVG, color-coded by node label (User/Prompt/Model/Agent/Tool/Dataset/Violation), risk
nodes glowing. A query selector runs the named queries; show the executed Cypher verbatim and
the tabular rows under the graph. Default view = `full-graph`; running `phi-to-unauthorized-ai`
highlights/animates the matching subgraph. Data comes LIVE from graph-svc — no canned layouts.

## Rules (unchanged)
Simulated telemetry labelled SIMULATED (R4). Control IDs only from fixtures (R5). No bespoke
detection paths. Every claim backed by run evidence.

## Verification Gate 3 (what I record)
Scenario 5 run → score trajectory 12→24→36→51→76, threshold breach at event 5, exactly ONE
violation row with agent_id set and a fixture-sourced control. The PHI graph query returning
REAL rows (nurse.joy@example.com → gemini, from Scenario 3 data). The Cypher + result pasted.
Animated graph + agents page screenshots.
