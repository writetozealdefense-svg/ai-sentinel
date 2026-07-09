package copilot

import (
	"strings"
	"testing"
	"time"

	"github.com/zealdefense/ai-sentinel/services/internal/compliance"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
)

// promptBundle is a synthetic scenario-1-shaped bundle: a BLOCK prompt with
// SECRET + PII findings mapped to a GDPR fixture control.
func promptBundle() *Bundle {
	return &Bundle{
		Violation: db.Violation{
			ID:          "11111111-1111-1111-1111-111111111111",
			PromptID:    "22222222-2222-2222-2222-222222222222",
			UserEmail:   "alice@corp.example",
			Framework:   "GDPR",
			ControlID:   "Art. 32",
			Category:    "SECRET",
			Severity:    "high",
			Description: "Security of processing — appropriate technical/organisational measures.",
			OccurredAt:  time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		},
		Prompt: &db.Prompt{
			ID:              "22222222-2222-2222-2222-222222222222",
			UserEmail:       "alice@corp.example",
			Provider:        "chatgpt",
			ArtifactName:    "creds.txt",
			Decision:        "BLOCK",
			RiskScore:       95,
			RedactedPreview: "here is the key <AWS_KEY> and mail <EMAIL_ADDRESS> " + strings.Repeat("x", 300),
			PolicyTrace:     "unsanctioned provider; SECRET category present",
			Scenario:        "scenario-1",
		},
		Findings: []db.Finding{
			{ID: "f-aws", Type: "AWS_KEY", Category: "SECRET", Engine: "yara", Score: 0.98, Occurrences: 2},
			{ID: "f-mail", Type: "EMAIL_ADDRESS", Category: "PII", Engine: "presidio", Score: 0.85, Occurrences: 3},
		},
		Sanctioned: false,
		Control: &ControlRef{
			Framework:   "GDPR",
			DisplayName: "EU General Data Protection Regulation",
			Control: compliance.Control{
				ControlID:        "Art. 32",
				Title:            "Security of processing",
				Description:      "Appropriate technical/organisational measures incl. encryption and access control.",
				MappedCategories: []string{"PII", "SECRET"},
				Severity:         "high",
				Review:           "# REVIEW: verify control ID",
			},
		},
	}
}

func agentBundle() *Bundle {
	return &Bundle{
		Violation: db.Violation{
			ID:          "33333333-3333-3333-3333-333333333333",
			AgentID:     "agent-orch-1",
			UserEmail:   "bob@corp.example",
			Framework:   "NIST_AI_RMF",
			ControlID:   "GOVERN-1.1",
			Category:    "OTHER",
			Severity:    "high",
			Description: "Agent risk threshold breached (score 76): 3 tools (jira, github, salesforce); privilege escalation",
			OccurredAt:  time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC),
		},
		Agent: &db.Agent{
			ID: "agent-orch-1", Name: "orchestrator", Framework: "langgraph",
			OwnerEmail: "bob@corp.example", RiskScore: 76, Simulated: true,
		},
		AgentRiskFactors: []string{"3 tools (jira, github, salesforce)", "privilege escalation"},
		Control: &ControlRef{
			Framework:   "NIST_AI_RMF",
			DisplayName: "NIST AI Risk Management Framework",
			Control: compliance.Control{
				ControlID: "GOVERN-1.1", Title: "Governance of AI risk",
				Description: "Policies and accountability for AI risk are in place.",
				MappedCategories: []string{"OTHER"}, Severity: "high",
				Review: "# REVIEW: verify control ID",
			},
		},
	}
}

func TestComposeProducesThreeViews(t *testing.T) {
	views, _ := Compose(promptBundle())
	if views.Executive == "" || views.Technical == "" || views.Compliance == "" {
		t.Fatalf("expected all three views non-empty, got %+v", views)
	}
}

// TestComposeFactsGrounded asserts that the facts rendered into each view are
// exactly the bundle's fields — user, provider, decision, control, categories,
// engines, counts, risk components, policy trace, redaction sample.
func TestComposeFactsGrounded(t *testing.T) {
	b := promptBundle()
	views, _ := Compose(b)

	execWant := []string{
		"alice@corp.example", "chatgpt", "unsanctioned", "BLOCK", `"creds.txt"`,
		"SECRET", "high-severity", "EU General Data Protection Regulation",
		"Art. 32", "Security of processing", "95/100", "2026-07-01T12:00:00Z",
	}
	for _, w := range execWant {
		if !strings.Contains(views.Executive, w) {
			t.Errorf("executive view missing %q:\n%s", w, views.Executive)
		}
	}

	techWant := []string{
		"AWS_KEY (SECRET) x2 via yara", "EMAIL_ADDRESS (PII) x3 via presidio",
		// SECRET severity weight 40; volume min(5,20)*2 = 10; confidence
		// round(0.98*15) = 15; unsanctioned 10 -> 75 (internal/risk formula).
		"severity 40 + volume 10 + confidence 15 + unsanctioned 10 = 75/100",
		"stored prompt risk score 95/100",
		"unsanctioned provider; SECRET category present", // OPA policy trace
	}
	for _, w := range techWant {
		if !strings.Contains(views.Technical, w) {
			t.Errorf("technical view missing %q:\n%s", w, views.Technical)
		}
	}
	// Redaction sample: first 200 chars of redacted_preview, then truncated.
	if !strings.Contains(views.Technical, b.Prompt.RedactedPreview[:200]) {
		t.Errorf("technical view missing 200-char redaction sample:\n%s", views.Technical)
	}
	if strings.Contains(views.Technical, b.Prompt.RedactedPreview) {
		t.Errorf("technical view quotes the FULL preview; want it truncated to ~200 chars")
	}

	compWant := []string{
		"EU General Data Protection Regulation", "GDPR", "Art. 32",
		`"Security of processing"`,
		"Appropriate technical/organisational measures incl. encryption and access control.",
		"PII, SECRET",                     // mappedCategories
		"# REVIEW: verify control ID",     // review marker verbatim
		"block recorded, violation logged, user coaching recommended",
	}
	for _, w := range compWant {
		if !strings.Contains(views.Compliance, w) {
			t.Errorf("compliance view missing %q:\n%s", w, views.Compliance)
		}
	}
}

// TestComposeNeverInventsControlText: with no fixture control matched, no
// control title/description may appear — only the violation row's own fields.
func TestComposeNeverInventsControlText(t *testing.T) {
	b := promptBundle()
	b.Control = nil
	views, cites := Compose(b)
	if strings.Contains(views.Compliance, "Security of processing") {
		t.Errorf("compliance view invented control text with no fixture match:\n%s", views.Compliance)
	}
	if !strings.Contains(views.Compliance, "not present in the loaded compliance fixtures") {
		t.Errorf("compliance view should state the control is not in fixtures:\n%s", views.Compliance)
	}
	for _, c := range cites {
		if c.Kind == "control" {
			t.Errorf("citations include a control citation with no fixture match: %+v", c)
		}
	}
}

func TestComposeCitations(t *testing.T) {
	b := promptBundle()
	_, cites := Compose(b)

	kinds := map[string]int{}
	byKindRef := map[string]string{}
	for _, c := range cites {
		kinds[c.Kind]++
		byKindRef[c.Kind] = c.Ref
	}
	if kinds["violation"] != 1 || byKindRef["violation"] != b.Violation.ID {
		t.Errorf("want exactly one violation citation ref=%s, got %+v", b.Violation.ID, cites)
	}
	if kinds["control"] != 1 || byKindRef["control"] != "Art. 32" {
		t.Errorf("want exactly one control citation ref=Art. 32, got %+v", cites)
	}
	if kinds["prompt"] != 1 || byKindRef["prompt"] != b.Prompt.ID {
		t.Errorf("want exactly one prompt citation ref=%s, got %+v", b.Prompt.ID, cites)
	}
	if kinds["finding"] != 2 {
		t.Errorf("want two finding citations, got %+v", cites)
	}
	for _, c := range cites {
		if c.Kind == "violation" && !strings.Contains(c.Excerpt, "Security of processing") {
			t.Errorf("violation citation excerpt should come from description, got %q", c.Excerpt)
		}
	}
}

func TestComposeAgentBundle(t *testing.T) {
	b := agentBundle()
	views, cites := Compose(b)

	if views.Executive == "" || views.Technical == "" || views.Compliance == "" {
		t.Fatalf("expected all three views non-empty, got %+v", views)
	}
	for _, w := range []string{"orchestrator", "agent-orch-1", "langgraph", "76/100", "bob@corp.example"} {
		if !strings.Contains(views.Executive, w) {
			t.Errorf("agent executive view missing %q:\n%s", w, views.Executive)
		}
	}
	for _, w := range []string{"3 tools (jira, github, salesforce)", "privilege escalation", "76/100", "threshold 70"} {
		if !strings.Contains(views.Technical, w) {
			t.Errorf("agent technical view missing %q:\n%s", w, views.Technical)
		}
	}
	if !strings.Contains(views.Compliance, "agent risk review recommended") {
		t.Errorf("agent compliance view missing remediation posture:\n%s", views.Compliance)
	}
	for _, c := range cites {
		if c.Kind == "prompt" || c.Kind == "finding" {
			t.Errorf("agent bundle must not cite prompts/findings, got %+v", c)
		}
	}
	kinds := map[string]bool{}
	for _, c := range cites {
		kinds[c.Kind] = true
	}
	if !kinds["violation"] || !kinds["control"] {
		t.Errorf("agent citations must include violation + control, got %+v", cites)
	}
}

func TestComposeFallback(t *testing.T) {
	views, cites := ComposeFallback()
	if views.Executive == "" {
		t.Fatal("fallback must carry an honest answer")
	}
	for _, w := range []string{"riskiest user", "riskiest agent", "compliance posture", "blocked"} {
		if !strings.Contains(views.Executive, w) {
			t.Errorf("fallback should list capability %q:\n%s", w, views.Executive)
		}
	}
	if cites == nil || len(cites) != 0 {
		t.Errorf("fallback citations must be empty-but-present, got %#v", cites)
	}
}

func TestComposeAggregates(t *testing.T) {
	u := &UserRiskSummary{
		Email: "alice@corp.example", ViolationCount: 7,
		Recent: []db.Violation{{ID: "v-1", Framework: "GDPR", ControlID: "Art. 32", Category: "SECRET", Severity: "high", Description: "d"}},
	}
	views, cites := ComposeRiskiestUser(u)
	if !strings.Contains(views.Executive, "alice@corp.example") || !strings.Contains(views.Executive, "7 recorded violations") {
		t.Errorf("riskiest-user text wrong:\n%s", views.Executive)
	}
	if len(cites) != 1 || cites[0].Kind != "violation" || cites[0].Ref != "v-1" {
		t.Errorf("riskiest-user citations wrong: %+v", cites)
	}

	a := &AgentRiskSummary{
		Agent:      db.Agent{ID: "agent-orch-1", Name: "orchestrator", Framework: "langgraph", RiskScore: 76, OwnerEmail: "bob@corp.example"},
		Factors:    []string{"privilege escalation"},
		Violations: []db.Violation{{ID: "v-2", Framework: "NIST_AI_RMF", ControlID: "GOVERN-1.1", Category: "OTHER", Severity: "high"}},
	}
	views, cites = ComposeRiskiestAgent(a)
	for _, w := range []string{"orchestrator", "langgraph", "76/100", "privilege escalation"} {
		if !strings.Contains(views.Executive, w) {
			t.Errorf("riskiest-agent text missing %q:\n%s", w, views.Executive)
		}
	}
	if len(cites) != 1 || cites[0].Ref != "v-2" {
		t.Errorf("riskiest-agent citations wrong: %+v", cites)
	}

	items := []FrameworkPosture{
		{Framework: "GDPR", OpenViolations: 34, Latest: &db.Violation{ID: "v-3", Framework: "GDPR", ControlID: "Art. 32", Category: "PII", Severity: "high"}},
		{Framework: "HIPAA", OpenViolations: 12},
	}
	views, cites = ComposeCompliancePosture(items)
	if !strings.Contains(views.Compliance, "GDPR: 34 open violations") || !strings.Contains(views.Compliance, "HIPAA: 12 open violations") {
		t.Errorf("posture text wrong:\n%s", views.Compliance)
	}
	if len(cites) != 1 || cites[0].Ref != "v-3" {
		t.Errorf("posture citations wrong: %+v", cites)
	}
}

func TestViewsFilter(t *testing.T) {
	v := Views{Executive: "e", Technical: "t", Compliance: "c"}
	if got := v.Filter("technical"); got.Technical != "t" || got.Executive != "" || got.Compliance != "" {
		t.Errorf("Filter(technical) = %+v", got)
	}
	if got := v.Filter("all"); got != v {
		t.Errorf("Filter(all) = %+v", got)
	}
	if got := v.Filter(""); got != v {
		t.Errorf("Filter(\"\") = %+v", got)
	}
}
