// Package copilot implements the AI Security Copilot (Phase 4): evidence-bundle
// retrieval over the platform's OWN data (RAG, not generation), a deterministic
// offline composer that renders three grounded narrative views per violation,
// a deterministic question router, and a minimal live-mode stub gated behind
// COPILOT_LIVE=true + COPILOT_API_KEY.
//
// Contract: contracts/openapi/copilot.yaml + docs/PHASE4-INTEGRATION.md.
// Every fact in a composed view comes from the evidence bundle; control text
// comes ONLY from the compliance fixtures (Brief R5).
package copilot

import (
	"github.com/zealdefense/ai-sentinel/services/internal/compliance"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
)

// Citation matches contracts/openapi/copilot.yaml Citation.
type Citation struct {
	Kind    string `json:"kind"` // violation | control | prompt | finding
	Ref     string `json:"ref"`
	Excerpt string `json:"excerpt,omitempty"`
}

// Views is the AskResponse views object. Absent views are omitted so a
// filtered response only carries the requested view.
type Views struct {
	Executive  string `json:"executive,omitempty"`
	Technical  string `json:"technical,omitempty"`
	Compliance string `json:"compliance,omitempty"`
}

// Filter returns only the requested view ("all" or "" keeps everything).
func (v Views) Filter(view string) Views {
	switch view {
	case "executive":
		return Views{Executive: v.Executive}
	case "technical":
		return Views{Technical: v.Technical}
	case "compliance":
		return Views{Compliance: v.Compliance}
	default: // "all" or ""
		return v
	}
}

// AskResponse matches contracts/openapi/copilot.yaml AskResponse.
type AskResponse struct {
	Mode      string     `json:"mode"` // offline | live
	Views     Views      `json:"views"`
	Citations []Citation `json:"citations"`
}

// ControlRef is the fixture control matched to a violation (framework +
// control_id), quoted verbatim by the composer — never invented.
type ControlRef struct {
	Framework   string
	DisplayName string
	Control     compliance.Control
}

// Bundle is the evidence bundle retrieved for one violation
// (PHASE4-INTEGRATION.md "Grounding = retrieval, not generation").
type Bundle struct {
	Violation db.Violation
	// Prompt violations (prompt_id set):
	Prompt     *db.Prompt
	Findings   []db.Finding
	Sanctioned bool // models.sanctioned for the prompt's provider
	// Agent violations (agent_id set, prompt_id NULL):
	Agent            *db.Agent
	AgentRiskFactors []string
	// The matched fixture control (nil when the violation's framework/control
	// pair is not present in the loaded fixtures).
	Control *ControlRef
}

// FindControl matches a violation's framework + control_id against the loaded
// fixtures. Returns nil when absent — the composer then quotes only the
// violation row rather than inventing control text.
func FindControl(set *compliance.Set, framework, controlID string) *ControlRef {
	if set == nil {
		return nil
	}
	fw, ok := set.Framework(framework)
	if !ok {
		return nil
	}
	for _, c := range fw.Controls {
		if c.ControlID == controlID {
			return &ControlRef{Framework: fw.Framework, DisplayName: fw.DisplayName, Control: c}
		}
	}
	return nil
}
