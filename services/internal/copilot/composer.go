package copilot

// Offline composer (DEFAULT mode, Brief R3): deterministic template composition
// over the evidence bundle. Three short grounded narrative views — executive /
// technical / compliance — built ONLY from retrieved fields. No free invention:
// every fact rendered here is read from the Bundle.

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/zealdefense/ai-sentinel/services/internal/agentrisk"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
	"github.com/zealdefense/ai-sentinel/services/internal/risk"
)

const previewSample = 200 // chars of redacted_preview quoted in the technical view

// Compose renders the three offline views plus citations from an evidence bundle.
func Compose(b *Bundle) (Views, []Citation) {
	return Views{
		Executive:  composeExecutive(b),
		Technical:  composeTechnical(b),
		Compliance: composeCompliance(b),
	}, composeCitations(b)
}

// frameworkLabel prefers the fixture display name, falling back to the
// violation row's framework code.
func frameworkLabel(b *Bundle) string {
	if b.Control != nil && b.Control.DisplayName != "" {
		return b.Control.DisplayName
	}
	return b.Violation.Framework
}

// composeExecutive: business risk framing — who, what data category, to which
// AI system, decision taken, compliance exposure, severity.
func composeExecutive(b *Bundle) string {
	v := b.Violation
	var s []string

	switch {
	case b.Prompt != nil:
		p := b.Prompt
		who := v.UserEmail
		if who == "" {
			who = p.UserEmail
		}
		standing := "an unsanctioned"
		if b.Sanctioned {
			standing = "a sanctioned"
		}
		s = append(s, fmt.Sprintf("%s submitted content to %s, %s AI system, and the platform decision was %s.", who, p.Provider, standing, p.Decision))
		if p.ArtifactName != "" {
			s = append(s, fmt.Sprintf("The artifact involved was %q.", p.ArtifactName))
		}
		s = append(s, fmt.Sprintf("The content exposed %s data, raising a %s-severity violation under %s%s.", v.Category, v.Severity, frameworkLabel(b), controlClause(b)))
		s = append(s, fmt.Sprintf("Prompt risk score: %d/100.", p.RiskScore))
	case b.Agent != nil:
		a := b.Agent
		label := a.ID
		if a.Name != "" {
			label = fmt.Sprintf("%s (%s)", a.Name, a.ID)
		}
		s = append(s, fmt.Sprintf("Agent %s on the %s framework breached the agent risk threshold with a risk score of %d/100.", label, a.Framework, a.RiskScore))
		if a.OwnerEmail != "" {
			s = append(s, fmt.Sprintf("The agent is owned by %s.", a.OwnerEmail))
		}
		s = append(s, fmt.Sprintf("This raised a %s-severity %s violation under %s%s.", v.Severity, v.Category, frameworkLabel(b), controlClause(b)))
	default:
		s = append(s, fmt.Sprintf("A %s-severity %s violation was recorded under %s%s.", v.Severity, v.Category, frameworkLabel(b), controlClause(b)))
	}
	if v.Description != "" {
		s = append(s, fmt.Sprintf("Violation record: %s", ensurePeriod(v.Description)))
	}
	s = append(s, fmt.Sprintf("Recorded at %s.", v.OccurredAt.UTC().Format(time.RFC3339)))
	return strings.Join(s, " ")
}

// controlClause renders " (control <id>: <title>)" when the fixture control
// was matched, "" otherwise (the control id from the row is still shown in the
// compliance view).
func controlClause(b *Bundle) string {
	if b.Control == nil {
		return fmt.Sprintf(" (control %s)", b.Violation.ControlID)
	}
	return fmt.Sprintf(" (control %s: %s)", b.Control.Control.ControlID, b.Control.Control.Title)
}

// composeTechnical: pipeline mechanics — detection engines + entity types and
// counts, the Phase-1 risk-score breakdown, the OPA policy trace, and the first
// ~200 chars of the redacted preview.
func composeTechnical(b *Bundle) string {
	var s []string

	switch {
	case b.Prompt != nil:
		p := b.Prompt
		if len(b.Findings) > 0 {
			parts := make([]string, 0, len(b.Findings))
			for _, f := range b.Findings {
				parts = append(parts, fmt.Sprintf("%s (%s) x%d via %s, score %.2f", f.Type, f.Category, f.Occurrences, f.Engine, f.Score))
			}
			s = append(s, "Detection findings: "+strings.Join(parts, "; ")+".")
		} else {
			s = append(s, "No detection findings are recorded for this prompt.")
		}
		s = append(s, riskBreakdown(b))
		if p.PolicyTrace != "" {
			s = append(s, "OPA policy trace: "+ensurePeriod(p.PolicyTrace))
		}
		if p.RedactedPreview != "" {
			sample := p.RedactedPreview
			if len(sample) > previewSample {
				sample = sample[:previewSample] + "…"
			}
			s = append(s, fmt.Sprintf("Redaction sample: %q", sample))
		}
	case b.Agent != nil:
		a := b.Agent
		if len(b.AgentRiskFactors) > 0 {
			s = append(s, "Agent risk factors: "+strings.Join(b.AgentRiskFactors, "; ")+".")
		}
		s = append(s, fmt.Sprintf("Agent risk score: %d/100 (violation threshold %d, per the Phase-3 agent risk formula).", a.RiskScore, agentrisk.Threshold))
		if b.Violation.Description != "" {
			s = append(s, "Violation record: "+ensurePeriod(b.Violation.Description))
		}
	default:
		s = append(s, fmt.Sprintf("Violation %s carries category %s and severity %s; no linked prompt or agent record is available.", b.Violation.ID, b.Violation.Category, b.Violation.Severity))
	}
	return strings.Join(s, " ")
}

// riskBreakdown recomputes the Phase-1 formula components
// (severity/volume/confidence/unsanctioned per internal/risk) from the
// aggregated findings, and shows the stored prompt score alongside.
func riskBreakdown(b *Bundle) string {
	cats := distinctCategories(b.Findings)
	total := 0
	highest := 0.0
	for _, f := range b.Findings {
		total += f.Occurrences
		if f.Score > highest {
			highest = f.Score
		}
	}
	sev := risk.SeverityWeight(cats)
	vol := total
	if vol > 20 {
		vol = 20
	}
	volume := vol * 2
	confidence := int(math.Round(highest * 15))
	unsanctioned := 10
	if b.Sanctioned {
		unsanctioned = 0
	}
	computed := risk.Score(risk.Input{Categories: cats, TotalFindings: total, HighestScore: highest, Sanctioned: b.Sanctioned})
	return fmt.Sprintf("Risk breakdown (Phase-1 formula): severity %d + volume %d + confidence %d + unsanctioned %d = %d/100; stored prompt risk score %d/100.",
		sev, volume, confidence, unsanctioned, computed, b.Prompt.RiskScore)
}

func distinctCategories(fs []db.Finding) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range fs {
		if !seen[f.Category] {
			seen[f.Category] = true
			out = append(out, f.Category)
		}
	}
	sort.Strings(out)
	return out
}

// composeCompliance: the fixture control quoted verbatim (framework, controlId,
// title, description), mapped categories, the REVIEW marker verbatim, and the
// remediation posture line.
func composeCompliance(b *Bundle) string {
	v := b.Violation
	var s []string

	if b.Control != nil {
		c := b.Control.Control
		s = append(s, fmt.Sprintf("%s (%s) control %s — %q: %s", b.Control.DisplayName, b.Control.Framework, c.ControlID, c.Title, ensurePeriod(c.Description)))
		if len(c.MappedCategories) > 0 {
			s = append(s, "Mapped categories: "+strings.Join(c.MappedCategories, ", ")+".")
		}
		if c.Review != "" {
			// Surfaced verbatim (Brief R5) — fixtures carry "# REVIEW ..." until verified.
			s = append(s, "Review marker: "+c.Review)
		}
	} else {
		s = append(s, fmt.Sprintf("Control %s of framework %s is not present in the loaded compliance fixtures; only the recorded violation is quoted.", v.ControlID, v.Framework))
	}
	s = append(s, fmt.Sprintf("This violation is recorded as category %s with severity %s.", v.Category, v.Severity))

	if b.Prompt != nil {
		s = append(s, fmt.Sprintf("Remediation posture: %s recorded, violation logged, user coaching recommended.", strings.ToLower(b.Prompt.Decision)))
	} else if b.Agent != nil {
		s = append(s, "Remediation posture: violation logged, agent risk review recommended.")
	} else {
		s = append(s, "Remediation posture: violation logged.")
	}
	return strings.Join(s, " ")
}

// composeCitations cites every record the views were grounded in: the
// violation row, the fixture control, and — when present — the prompt row and
// each aggregated finding.
func composeCitations(b *Bundle) []Citation {
	v := b.Violation
	out := []Citation{}

	excerpt := v.Description
	if excerpt == "" {
		excerpt = fmt.Sprintf("%s %s violation (%s/%s)", v.Severity, v.Category, v.Framework, v.ControlID)
	}
	out = append(out, Citation{Kind: "violation", Ref: v.ID, Excerpt: truncate(excerpt, 200)})

	if b.Control != nil {
		c := b.Control.Control
		out = append(out, Citation{Kind: "control", Ref: c.ControlID, Excerpt: truncate(c.Title+" — "+c.Description, 200)})
	}
	if b.Prompt != nil {
		p := b.Prompt
		out = append(out, Citation{
			Kind: "prompt", Ref: p.ID,
			Excerpt: fmt.Sprintf("%s -> %s: decision %s, risk %d/100", p.UserEmail, p.Provider, p.Decision, p.RiskScore),
		})
		for _, f := range b.Findings {
			out = append(out, Citation{
				Kind: "finding", Ref: f.ID,
				Excerpt: fmt.Sprintf("%s (%s) x%d via %s", f.Type, f.Category, f.Occurrences, f.Engine),
			})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Deterministic aggregate answers (POST /copilot/ask without violationId)
// ---------------------------------------------------------------------------

// UserRiskSummary is the "riskiest user" aggregate: the user with the most
// recorded (open) violations plus their most recent violation rows.
type UserRiskSummary struct {
	Email          string
	ViolationCount int
	Recent         []db.Violation // newest first, capped by the retriever
}

// ComposeRiskiestUser renders the riskiest-user aggregate deterministically.
func ComposeRiskiestUser(u *UserRiskSummary) (Views, []Citation) {
	text := fmt.Sprintf("The riskiest user by open violation count is %s with %d recorded violations.", u.Email, u.ViolationCount)
	cites := []Citation{}
	for i, v := range u.Recent {
		if i == 0 {
			text += fmt.Sprintf(" Most recent: %s-severity %s violation under %s control %s.", v.Severity, v.Category, v.Framework, v.ControlID)
		}
		cites = append(cites, Citation{Kind: "violation", Ref: v.ID, Excerpt: truncate(violationExcerpt(v), 200)})
	}
	return Views{Executive: text}, cites
}

// AgentRiskSummary is the "riskiest agent" aggregate: the agent with the top
// risk score, its recomputed risk factors, and its violation rows.
type AgentRiskSummary struct {
	Agent      db.Agent
	Factors    []string
	Violations []db.Violation // newest first, capped by the retriever
}

// ComposeRiskiestAgent renders the riskiest-agent aggregate deterministically.
func ComposeRiskiestAgent(a *AgentRiskSummary) (Views, []Citation) {
	label := a.Agent.ID
	if a.Agent.Name != "" {
		label = fmt.Sprintf("%s (%s)", a.Agent.Name, a.Agent.ID)
	}
	text := fmt.Sprintf("The riskiest agent is %s on the %s framework with a risk score of %d/100.", label, a.Agent.Framework, a.Agent.RiskScore)
	if a.Agent.OwnerEmail != "" {
		text += fmt.Sprintf(" It is owned by %s.", a.Agent.OwnerEmail)
	}
	if len(a.Factors) > 0 {
		text += " Risk factors: " + strings.Join(a.Factors, "; ") + "."
	}
	cites := []Citation{}
	for _, v := range a.Violations {
		cites = append(cites, Citation{Kind: "violation", Ref: v.ID, Excerpt: truncate(violationExcerpt(v), 200)})
	}
	return Views{Executive: text}, cites
}

// FrameworkPosture is one framework's open-violation count plus its most
// recent violation (cited as evidence).
type FrameworkPosture struct {
	Framework      string
	OpenViolations int
	Latest         *db.Violation
}

// ComposeCompliancePosture renders the per-framework open-violation summary.
func ComposeCompliancePosture(items []FrameworkPosture) (Views, []Citation) {
	parts := make([]string, 0, len(items))
	cites := []Citation{}
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%s: %d open violations", it.Framework, it.OpenViolations))
		if it.Latest != nil {
			cites = append(cites, Citation{Kind: "violation", Ref: it.Latest.ID, Excerpt: truncate(violationExcerpt(*it.Latest), 200)})
		}
	}
	text := "Compliance posture across recorded violations — " + strings.Join(parts, "; ") + "."
	return Views{Compliance: text}, cites
}

// ComposeFallback is the honest unknown-intent answer: it lists what the
// copilot CAN answer and cites nothing (empty-but-present citations array).
func ComposeFallback() (Views, []Citation) {
	text := "I can only answer questions grounded in this platform's recorded data. Try one of: " +
		"\"why was the last prompt blocked\", " +
		"\"who is the riskiest user\", " +
		"\"which is the riskiest agent\", " +
		"\"what is our compliance posture\" — " +
		"or include a violationId to have a specific violation explained."
	return Views{Executive: text}, []Citation{}
}

func violationExcerpt(v db.Violation) string {
	if v.Description != "" {
		return v.Description
	}
	return fmt.Sprintf("%s %s violation (%s/%s)", v.Severity, v.Category, v.Framework, v.ControlID)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func ensurePeriod(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return t
	}
	switch t[len(t)-1] {
	case '.', '!', '?':
		return t
	}
	return t + "."
}
