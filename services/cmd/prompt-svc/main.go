// Command prompt-svc is the inspection orchestrator (Interface 3). Port 8082.
//
// POST /prompts/inspect runs: discovery record -> detection scan -> risk score
// -> OPA decision -> persist prompts+findings -> publish ai.prompt.inspected ->
// (on REDACT/BLOCK) map categories to frameworks, insert one violation per
// framework + publish ai.violation.raised -> return InspectResult.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zealdefense/ai-sentinel/services/internal/bus"
	"github.com/zealdefense/ai-sentinel/services/internal/compliance"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
	"github.com/zealdefense/ai-sentinel/services/internal/detect"
	"github.com/zealdefense/ai-sentinel/services/internal/httpx"
	"github.com/zealdefense/ai-sentinel/services/internal/opa"
	"github.com/zealdefense/ai-sentinel/services/internal/risk"
)

const serviceName = "prompt-svc"

type server struct {
	db     *db.DB
	bus    *bus.Bus
	detect *detect.Client
	opa    *opa.Client
	comp   *compliance.Set
}

func main() {
	ctx := context.Background()

	databaseURL := httpx.Env(os.Getenv, "DATABASE_URL", "postgres://sentinel:sentinel_dev_pw@localhost:5432/sentinel?sslmode=disable")
	natsURL := httpx.Env(os.Getenv, "NATS_URL", "nats://localhost:4222")
	detectionURL := httpx.Env(os.Getenv, "DETECTION_URL", "http://localhost:8000")
	opaURL := httpx.Env(os.Getenv, "OPA_URL", "http://localhost:8181")
	fixturesDir := httpx.Env(os.Getenv, "COMPLIANCE_FIXTURES", "/fixtures/compliance")
	addr := ":" + httpx.Env(os.Getenv, "PORT", "8082")

	comp, err := compliance.Load(fixturesDir)
	if err != nil {
		log.Fatalf("[%s] load compliance fixtures from %s: %v", serviceName, fixturesDir, err)
	}
	log.Printf("[%s] loaded %d compliance frameworks", serviceName, len(comp.Frameworks()))

	database, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("[%s] db connect: %v", serviceName, err)
	}
	defer database.Close()

	b, _ := bus.Connect(natsURL, serviceName) // fail-soft
	defer b.Close()

	s := &server{
		db:     database,
		bus:    b,
		detect: detect.New(detectionURL),
		opa:    opa.New(opaURL),
		comp:   comp,
	}

	r := httpx.NewRouter(serviceName)
	r.POST("/prompts/inspect", s.inspect)
	r.GET("/prompts", s.listPrompts)
	r.GET("/prompts/:id", s.getPrompt)

	log.Printf("[%s] listening on %s", serviceName, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[%s] run: %v", serviceName, err)
	}
}

// --- DTOs (contracts/openapi/prompts.yaml) ---

type inspectRequest struct {
	UserEmail    string `json:"userEmail"`
	Provider     string `json:"provider"`
	Content      string `json:"content"`
	ArtifactName string `json:"artifactName"`
	ContentType  string `json:"contentType"`
	Scenario     string `json:"scenario"`
}

type findingDTO struct {
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Engine   string  `json:"engine"`
	Score    float64 `json:"score"`
	Count    int     `json:"count"`
}

type inspectResult struct {
	PromptID        string       `json:"promptId"`
	UserEmail       string       `json:"userEmail"`
	Provider        string       `json:"provider"`
	ArtifactName    string       `json:"artifactName,omitempty"`
	Decision        string       `json:"decision"`
	RiskScore       int          `json:"riskScore"`
	Findings        []findingDTO `json:"findings"`
	RedactedPreview string       `json:"redactedPreview,omitempty"`
	ViolationID     string       `json:"violationId,omitempty"`
	PolicyTrace     string       `json:"policyTrace,omitempty"`
	Timestamp       time.Time    `json:"timestamp"`
}

func (s *server) inspect(c *gin.Context) {
	ctx := c.Request.Context()

	var req inspectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	if req.UserEmail == "" || req.Provider == "" || req.Content == "" {
		httpx.Error(c, http.StatusBadRequest, "invalid request", "userEmail, provider, content are required")
		return
	}
	if req.ContentType == "" {
		req.ContentType = "text"
	}

	// 1. Fire-and-record SIMULATED discovery event (insert + publish).
	s.recordDiscovery(ctx, req)

	// 2. Ensure provider exists in models; read sanctioned flag (default false).
	sanctioned, err := s.db.EnsureModel(ctx, req.Provider)
	if err != nil {
		log.Printf("[%s] ensure model: %v", serviceName, err)
		// non-fatal: treat as unsanctioned
		sanctioned = false
	}

	// 3. Call detection-svc /scan.
	scan, err := s.detect.Scan(ctx, req.Content, req.ContentType, req.ArtifactName)
	if err != nil {
		httpx.Error(c, http.StatusBadGateway, "detection failed", err.Error())
		return
	}
	categories := scan.Categories()

	// 4. Compute deterministic risk score.
	score := risk.Score(risk.Input{
		Categories:    categories,
		TotalFindings: scan.Stats.TotalFindings,
		HighestScore:  scan.Stats.HighestScore,
		Sanctioned:    sanctioned,
	})

	// 5. Call OPA -> {action, reasons}; action maps directly to decision.
	dec, err := s.opa.Decide(ctx, opa.Input{
		Provider:      req.Provider,
		Sanctioned:    sanctioned,
		Categories:    categories,
		TotalFindings: scan.Stats.TotalFindings,
		HighestScore:  scan.Stats.HighestScore,
		ByCategory:    scan.Stats.ByCategory,
	})
	if err != nil {
		httpx.Error(c, http.StatusBadGateway, "policy decision failed", err.Error())
		return
	}
	decision := normalizeDecision(dec.Action)
	policyTrace := strings.Join(dec.Reasons, "; ")

	// 6. Persist prompts row + aggregated findings rows.
	promptID, err := s.db.InsertPrompt(ctx, db.Prompt{
		UserEmail:       req.UserEmail,
		Provider:        req.Provider,
		ArtifactName:    req.ArtifactName,
		ContentType:     req.ContentType,
		Decision:        decision,
		RiskScore:       score,
		RedactedPreview: scan.RedactedPreview,
		PolicyTrace:     policyTrace,
		Scenario:        req.Scenario,
	})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "persist failed", err.Error())
		return
	}

	agg := aggregateFindings(scan.Findings)
	if err := s.db.InsertFindings(ctx, promptID, agg); err != nil {
		log.Printf("[%s] insert findings: %v", serviceName, err)
	}

	// 7. Publish ai.prompt.inspected.
	now := time.Now().UTC()
	s.bus.Publish(bus.SubjectPromptInspected, map[string]any{
		"eventId":      uuid.NewString(),
		"promptId":     promptID,
		"userEmail":    req.UserEmail,
		"provider":     req.Provider,
		"artifactName": req.ArtifactName,
		"decision":     decision,
		"riskScore":    score,
		"categories":   categories,
		"findingCount": scan.Stats.TotalFindings,
		"scenario":     req.Scenario,
		"timestamp":    now.Format(time.RFC3339),
	})

	// 8. On REDACT/BLOCK, emit one violation per matching framework.
	var firstViolationID string
	if decision == "REDACT" || decision == "BLOCK" {
		firstViolationID = s.emitViolations(ctx, promptID, req.UserEmail, categories)
	}

	// 9. Return InspectResult.
	c.JSON(http.StatusOK, inspectResult{
		PromptID:        promptID,
		UserEmail:       req.UserEmail,
		Provider:        req.Provider,
		ArtifactName:    req.ArtifactName,
		Decision:        decision,
		RiskScore:       score,
		Findings:        toFindingDTOs(agg),
		RedactedPreview: scan.RedactedPreview,
		ViolationID:     firstViolationID,
		PolicyTrace:     policyTrace,
		Timestamp:       now,
	})
}

// recordDiscovery inserts a simulated discovery_events row (action=upload) and
// publishes ai.discovery.observed. Failures are logged, never fatal.
func (s *server) recordDiscovery(ctx context.Context, req inspectRequest) {
	occurred := time.Now().UTC()
	id, err := s.db.InsertDiscoveryEvent(ctx, db.DiscoveryEvent{
		Provider:     req.Provider,
		UserEmail:    req.UserEmail,
		Source:       "endpoint",
		Action:       "upload",
		ArtifactName: req.ArtifactName,
		Bytes:        int64(len(req.Content)),
		Simulated:    true,
		OccurredAt:   occurred,
	})
	if err != nil {
		log.Printf("[%s] record discovery: %v", serviceName, err)
		return
	}
	s.bus.Publish(bus.SubjectDiscoveryObserved, map[string]any{
		"eventId":      id,
		"provider":     req.Provider,
		"userEmail":    req.UserEmail,
		"source":       "endpoint",
		"action":       "upload",
		"artifactName": req.ArtifactName,
		"bytes":        len(req.Content),
		"simulated":    true,
		"timestamp":    occurred.Format(time.RFC3339),
	})
}

// emitViolations maps categories to frameworks via the fixtures, inserts one
// violation per matching framework, publishes ai.violation.raised per row, and
// returns the FIRST inserted violation id (for the synchronous API caller).
func (s *server) emitViolations(ctx context.Context, promptID, userEmail string, categories []string) string {
	matches := s.comp.MatchFrameworks(categories)
	var first string
	for _, m := range matches {
		desc := m.Control.Title
		if m.Control.Description != "" {
			desc = m.Control.Title + " — " + m.Control.Description
		}
		vid, err := s.db.InsertViolation(ctx, db.Violation{
			PromptID:    promptID,
			UserEmail:   userEmail,
			Framework:   m.Framework,
			ControlID:   m.Control.ControlID,
			Category:    m.Category,
			Severity:    m.Control.Severity,
			Description: desc,
		})
		if err != nil {
			log.Printf("[%s] insert violation (%s): %v", serviceName, m.Framework, err)
			continue
		}
		if first == "" {
			first = vid
		}
		s.bus.Publish(bus.SubjectViolationRaised, map[string]any{
			"eventId":     uuid.NewString(),
			"violationId": vid,
			"promptId":    promptID,
			"userEmail":   userEmail,
			"framework":   m.Framework,
			"controlId":   m.Control.ControlID,
			"category":    m.Category,
			"severity":    m.Control.Severity,
			"description": desc,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		})
	}
	return first
}

// promptSummaryDTO matches contracts/openapi/prompts.yaml PromptSummary.
type promptSummaryDTO struct {
	PromptID     string    `json:"promptId"`
	UserEmail    string    `json:"userEmail"`
	Provider     string    `json:"provider"`
	ArtifactName string    `json:"artifactName,omitempty"`
	Decision     string    `json:"decision"`
	RiskScore    int       `json:"riskScore"`
	Categories   []string  `json:"categories"`
	Scenario     string    `json:"scenario,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// listPrompts serves GET /prompts?limit=&decision= — newest-first PromptSummary
// rows for SOC live monitoring. limit defaults to 50, max 500; decision is an
// optional ALLOW/WARN/REDACT/BLOCK filter.
func (s *server) listPrompts(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}
	decision := normalizeDecisionFilter(c.Query("decision"))

	rows, err := s.db.ListPrompts(c.Request.Context(), decision, limit)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	out := make([]promptSummaryDTO, 0, len(rows))
	for _, p := range rows {
		cats := p.Categories
		if cats == nil {
			cats = []string{}
		}
		out = append(out, promptSummaryDTO{
			PromptID:     p.ID,
			UserEmail:    p.UserEmail,
			Provider:     p.Provider,
			ArtifactName: p.ArtifactName,
			Decision:     p.Decision,
			RiskScore:    p.RiskScore,
			Categories:   cats,
			Scenario:     p.Scenario,
			Timestamp:    p.OccurredAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

// normalizeDecisionFilter upper-cases a decision query param, returning "" (no
// filter) when empty or not one of the four canonical decisions.
func normalizeDecisionFilter(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ALLOW", "WARN", "REDACT", "BLOCK":
		return strings.ToUpper(strings.TrimSpace(s))
	default:
		return ""
	}
}

func (s *server) getPrompt(c *gin.Context) {
	id := c.Param("id")
	p, fs, err := s.db.GetPrompt(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(c, http.StatusNotFound, "not found", "no prompt with id "+id)
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	findings := make([]findingDTO, 0, len(fs))
	for _, f := range fs {
		findings = append(findings, findingDTO{
			Type: f.Type, Category: f.Category, Engine: f.Engine, Score: f.Score, Count: f.Occurrences,
		})
	}
	c.JSON(http.StatusOK, inspectResult{
		PromptID:        p.ID,
		UserEmail:       p.UserEmail,
		Provider:        p.Provider,
		ArtifactName:    p.ArtifactName,
		Decision:        p.Decision,
		RiskScore:       p.RiskScore,
		Findings:        findings,
		RedactedPreview: p.RedactedPreview,
		PolicyTrace:     p.PolicyTrace,
		Timestamp:       p.OccurredAt,
	})
}

// aggregateFindings groups detection findings by type, keeping the max score and
// counting occurrences (matches the findings table semantics).
func aggregateFindings(fs []detect.Finding) []db.Finding {
	type key struct{ typ, cat, eng string }
	order := []key{}
	agg := map[key]*db.Finding{}
	for _, f := range fs {
		k := key{f.Type, f.Category, f.Engine}
		cur, ok := agg[k]
		if !ok {
			rule := ""
			if f.RuleName != nil {
				rule = *f.RuleName
			}
			agg[k] = &db.Finding{
				Type: f.Type, Category: f.Category, Engine: f.Engine,
				Score: f.Score, RuleName: rule, Occurrences: 1,
			}
			order = append(order, k)
			continue
		}
		cur.Occurrences++
		if f.Score > cur.Score {
			cur.Score = f.Score
		}
	}
	out := make([]db.Finding, 0, len(order))
	for _, k := range order {
		out = append(out, *agg[k])
	}
	return out
}

func toFindingDTOs(fs []db.Finding) []findingDTO {
	out := make([]findingDTO, 0, len(fs))
	for _, f := range fs {
		out = append(out, findingDTO{
			Type: f.Type, Category: f.Category, Engine: f.Engine, Score: f.Score, Count: f.Occurrences,
		})
	}
	return out
}

// normalizeDecision maps an OPA action to a canonical decision, defaulting to ALLOW.
func normalizeDecision(action string) string {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "BLOCK":
		return "BLOCK"
	case "REDACT":
		return "REDACT"
	case "WARN":
		return "WARN"
	default:
		return "ALLOW"
	}
}
