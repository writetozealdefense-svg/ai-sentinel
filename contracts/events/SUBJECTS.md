# NATS Subjects & Event Bus Contract

All services publish/consume against these subjects and the JSON Schemas in
`contracts/events/*.schema.json`. Every event carries `eventId`, `timestamp`, and (for
telemetry-derived events) a `simulated: true` flag per Brief R4.

| Subject                       | Publisher              | Consumers                          | Schema                          |
|-------------------------------|------------------------|------------------------------------|---------------------------------|
| `ai.discovery.observed`       | telemetry-generator → discovery-svc | discovery-svc, graph-svc | `discovery.observed.schema.json` |
| `ai.prompt.inspected`         | prompt-svc             | risk-svc, graph-svc, compliance-svc | `prompt.inspected.schema.json`  |
| `ai.dlp.scanned`              | dlp-svc                | risk-svc                            | `dlp.scanned.schema.json`       |
| `ai.agent.activity`           | telemetry-generator → agent-svc | agent-svc, graph-svc      | `agent.activity.schema.json`    |
| `ai.violation.raised`         | prompt-svc / agent-svc | compliance-svc, graph-svc, copilot-svc | `violation.raised.schema.json` |

## Subject naming rules
- Root namespace: `ai.`
- Domain segment: `discovery | prompt | dlp | agent | violation`
- Verb segment (past tense): `observed | inspected | scanned | activity | raised`
- Wildcard subscriptions: a service may subscribe to `ai.violation.>` etc.

## Delivery
- Compose path: core NATS (at-most-once). JetStream is enabled in `deploy/compose` for the
  `VIOLATIONS` stream so violations survive a consumer restart during a demo.
- All payloads are UTF-8 JSON validated against the matching schema before publish.
