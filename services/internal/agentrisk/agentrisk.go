// Package agentrisk implements the deterministic Agent Risk Score from
// PHASE3-INTEGRATION.md Interface 1:
//
//	+12 per distinct tool used                        (cap 36)
//	+15 per distinct sensitive data_access (target)   (cap 30)
//	+25 if any privilege_escalation event
//	+10 if any external_call event
//	score = clamp(sum, 0, 100)
//
// Sensitive data accesses are counted per distinct target (like tools are
// counted distinct) so the score is a pure function of WHAT the agent touched,
// not how many times telemetry replayed it — re-running a scenario leaves the
// score unchanged (idempotent threshold, no double-billing duplicate events).
//
// Threshold semantics: a violation is emitted only on the event that moves the
// score from <70 to >=70 (the crossing), never again for later events.
package agentrisk

import (
	"fmt"
	"strings"
)

// Threshold is the risk score at/above which an agent breaches (contract: 70).
const Threshold = 70

// Actions (contracts/openapi/agents.yaml AgentTelemetry.action).
const (
	ActionToolCall            = "tool_call"
	ActionDataAccess          = "data_access"
	ActionPrivilegeEscalation = "privilege_escalation"
	ActionExternalCall        = "external_call"
)

// Scoring weights and caps from the contract table.
const (
	perToolPoints      = 12
	toolCap            = 36
	perSensitivePoints = 15
	sensitiveCap       = 30
	privEscPoints      = 25
	externalPoints     = 10
)

// Event is the slice of an agent_activity row the formula needs.
type Event struct {
	Action    string
	Tool      string
	Target    string
	Sensitive bool
}

// Score computes the deterministic 0-100 risk score plus human-readable risk
// factors from an agent's full activity history. It is a pure function: the
// same history always yields the same score and factors.
func Score(history []Event) (int, []string) {
	var tools []string // distinct tools in first-seen order
	seen := map[string]bool{}
	var sensitiveTargets []string // distinct sensitive targets in first-seen order
	seenTarget := map[string]bool{}
	privEsc, external := false, false

	for _, e := range history {
		if e.Tool != "" && !seen[e.Tool] {
			seen[e.Tool] = true
			tools = append(tools, e.Tool)
		}
		switch e.Action {
		case ActionDataAccess:
			if e.Sensitive {
				t := e.Target
				if t == "" {
					t = "unspecified"
				}
				if !seenTarget[t] {
					seenTarget[t] = true
					sensitiveTargets = append(sensitiveTargets, t)
				}
			}
		case ActionPrivilegeEscalation:
			privEsc = true
		case ActionExternalCall:
			external = true
		}
	}

	sum := 0
	var factors []string

	if n := len(tools); n > 0 {
		pts := n * perToolPoints
		if pts > toolCap {
			pts = toolCap
		}
		sum += pts
		noun := "tools"
		if n == 1 {
			noun = "tool"
		}
		factors = append(factors, fmt.Sprintf("%d %s (%s)", n, noun, strings.Join(tools, ", ")))
	}
	if len(sensitiveTargets) > 0 {
		pts := len(sensitiveTargets) * perSensitivePoints
		if pts > sensitiveCap {
			pts = sensitiveCap
		}
		sum += pts
		factors = append(factors, fmt.Sprintf("sensitive data access (%s)", strings.Join(sensitiveTargets, ", ")))
	}
	if privEsc {
		sum += privEscPoints
		factors = append(factors, "privilege escalation")
	}
	if external {
		sum += externalPoints
		factors = append(factors, "external call")
	}

	if sum > 100 {
		sum = 100
	}
	if sum < 0 {
		sum = 0
	}
	return sum, factors
}

// Crossed reports whether an event moved the score across the threshold:
// previous score strictly below 70 and new score at/above 70. Events that keep
// the score at/above the threshold after a crossing return false, so exactly
// one violation is emitted per breach.
func Crossed(prev, curr int) bool {
	return prev < Threshold && curr >= Threshold
}
