package copilot

// Deterministic intent router for POST /copilot/ask without a violationId
// (PHASE4-INTEGRATION.md "Endpoints"). Pure string matching — no model, no
// randomness — so the same question always routes the same way.

import "strings"

// Intent is a routed question intent.
type Intent string

// Router intents.
const (
	IntentWhyBlocked        Intent = "why-blocked"        // ground on the most recent BLOCK-decision violation
	IntentRiskiestUser      Intent = "riskiest-user"      // top user by open violation count
	IntentRiskiestAgent     Intent = "riskiest-agent"     // top agent by risk_score
	IntentCompliancePosture Intent = "compliance-posture" // per-framework open-violation counts
	IntentUnknown           Intent = "unknown"            // honest fallback
)

// Route matches a free-text question to an intent. Order matters: the more
// specific aggregate intents are checked before the broad block/"why" match.
// Riskiest matching is word-order independent ("riskiest agent" and
// "which agent is the riskiest" both route) — the risk word and the subject
// word just have to co-occur.
func Route(question string) Intent {
	q := strings.ToLower(question)
	risky := strings.Contains(q, "riskiest") || strings.Contains(q, "risky") ||
		strings.Contains(q, "highest risk") || strings.Contains(q, "most risk")
	switch {
	case risky && strings.Contains(q, "user"):
		return IntentRiskiestUser
	case risky && strings.Contains(q, "agent"):
		return IntentRiskiestAgent
	case strings.Contains(q, "compliance") || strings.Contains(q, "posture"):
		return IntentCompliancePosture
	case strings.Contains(q, "block") || strings.Contains(q, "why"):
		// "blocked" contains "block".
		return IntentWhyBlocked
	default:
		return IntentUnknown
	}
}
