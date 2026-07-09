package copilot

import "testing"

func TestRoute(t *testing.T) {
	cases := []struct {
		question string
		want     Intent
	}{
		{"why was the last prompt blocked", IntentWhyBlocked},
		{"Why was this blocked?", IntentWhyBlocked},
		{"WHY did that happen", IntentWhyBlocked},
		{"what got blocked today", IntentWhyBlocked},
		{"who is the riskiest user", IntentRiskiestUser},
		{"Show me the RISKIEST USER this week", IntentRiskiestUser},
		{"which agent is the riskiest agent", IntentRiskiestAgent},
		{"riskiest agent?", IntentRiskiestAgent},
		// Word-order independent (the UI's suggested-question phrasings).
		{"Which agent is the riskiest?", IntentRiskiestAgent},
		{"which user is the most risky", IntentRiskiestUser},
		{"which agent carries the highest risk", IntentRiskiestAgent},
		{"what is our compliance posture", IntentCompliancePosture},
		{"Compliance Posture summary please", IntentCompliancePosture},
		// Specific aggregates win over the broad "why"/"block" match.
		{"why is bob the riskiest user", IntentRiskiestUser},
		{"why is our compliance posture bad", IntentCompliancePosture},
		// Unknown -> honest fallback, never hallucination.
		{"what's the weather in Riyadh", IntentUnknown},
		{"tell me a joke", IntentUnknown},
		{"", IntentUnknown},
	}
	for _, c := range cases {
		if got := Route(c.question); got != c.want {
			t.Errorf("Route(%q) = %s, want %s", c.question, got, c.want)
		}
	}
}
