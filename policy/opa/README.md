# Sentinel DLP — OPA Policy

Package: `sentinel.dlp`  
Rule: `decision`

## Query path

```
POST http://opa:8181/v1/data/sentinel/dlp/decision
Content-Type: application/json

{
  "input": {
    "provider":      "chatgpt",
    "sanctioned":    false,
    "categories":    ["PII"],
    "totalFindings": 40,
    "highestScore":  0.99,
    "byCategory":    {"PII": 40}
  }
}
```

## Response shape

```json
{
  "result": {
    "action":  "ALLOW | WARN | REDACT | BLOCK",
    "reasons": ["<human-readable explanation>", "..."]
  }
}
```

`result` is the value of `data.sentinel.dlp.decision`.

## Decision rules (precedence: BLOCK > REDACT > WARN > ALLOW)

| Action | Condition |
|--------|-----------|
| BLOCK  | `SECRET`, `PHI`, or `PCI` in `categories`; OR `PII` in `categories` AND `totalFindings >= 5` |
| REDACT | `PII` in `categories` AND `totalFindings < 5` (and not BLOCK) |
| WARN   | `CODE` in `categories` (and not BLOCK or REDACT) |
| ALLOW  | none of the above triggered |

`reasons` is always a non-empty array of strings that mention the triggering category/count and whether the provider is sanctioned.

## Running tests

```bash
docker run --rm \
  -v "<absolute-path-to>/policy/opa:/policy" \
  openpolicyagent/opa:latest \
  test /policy -v
```

## Ad-hoc evaluation

Write the input to a file (e.g. `input.json`) inside `policy/opa/`, then:

```bash
docker run --rm \
  -v "<absolute-path-to>/policy/opa:/policy" \
  openpolicyagent/opa:latest \
  eval -d /policy 'data.sentinel.dlp.decision' \
  -i /policy/input.json --format pretty
```
