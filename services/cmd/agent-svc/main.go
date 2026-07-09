// Command agent-svc is the agentic-AI security service (Phase 3 Interface 1).
// Port 8083, contract contracts/openapi/agents.yaml.
//
// POST /agents/telemetry ingests a SIMULATED agent activity event: upserts the
// agents row, inserts agent_activity, recomputes the deterministic 0-100 risk
// score (internal/agentrisk), publishes ai.agent.activity, and — only on the
// event that moves the score from <70 to >=70 — emits ONE ai.violation.raised
// mapped from category OTHER via the compliance fixtures (Brief R5: control ID
// from fixtures only, never hardcoded).
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

	"github.com/zealdefense/ai-sentinel/services/internal/agentrisk"
	"github.com/zealdefense/ai-sentinel/services/internal/bus"
	"github.com/zealdefense/ai-sentinel/services/internal/compliance"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
	"github.com/zealdefense/ai-sentinel/services/internal/httpx"
)

const serviceName = "agent-svc"

var validFrameworks = map[string]bool{
	"langgraph": true, "crewai": true, "autogen": true, "openai-agents-sdk": true, "mcp": true,
}

var validActions = map[string]bool{
	agentrisk.ActionToolCall:            true,
	agentrisk.ActionDataAccess:          true,
	agentrisk.ActionPrivilegeEscalation: true,
	agentrisk.ActionExternalCall:        true,
}

type server struct {
	db   *db.DB
	bus  *bus.Bus
	comp *compliance.Set
}

func main() {
	ctx := context.Background()

	databaseURL := httpx.Env(os.Getenv, "DATABASE_URL", "postgres://sentinel:sentinel_dev_pw@localhost:5432/sentinel?sslmode=disable")
	natsURL := httpx.Env(os.Getenv, "NATS_URL", "nats://localhost:4222")
	fixturesDir := httpx.Env(os.Getenv, "COMPLIANCE_FIXTURES", "/fixtures/compliance")
	addr := ":" + httpx.Env(os.Getenv, "PORT", "8083")

	set, err := compliance.Load(fixturesDir)
	if err != nil {
		log.Fatalf("[%s] load fixtures from %s: %v", serviceName, fixturesDir, err)
	}
	// Resolve the OTHER-category mapping now so a missing fixture fails fast at
	// startup instead of at threshold-crossing time.
	if m := set.MatchFrameworks([]string{"OTHER"}); len(m) == 0 {
		log.Fatalf("[%s] no framework in %s maps category OTHER — cannot emit threshold violations", serviceName, fixturesDir)
	}
	log.Printf("[%s] loaded %d frameworks from %s", serviceName, len(set.Frameworks()), fixturesDir)

	database, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("[%s] db connect: %v", serviceName, err)
	}
	defer database.Close()

	b, _ := bus.Connect(natsURL, serviceName) // fail-soft: publishes degrade to no-ops
	defer b.Close()

	s := &server{db: database, bus: b, comp: set}

	r := httpx.NewRouter(serviceName)
	r.POST("/agents/telemetry", s.ingestTelemetry)
	r.GET("/agents", s.listAgents)
	r.GET("/agents/:id", s.getAgent)

	log.Printf("[%s] listening on %s", serviceName, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[%s] run: %v", serviceName, err)
	}
}

// telemetryRequest matches contracts/openapi/agents.yaml AgentTelemetry plus the
// optional owner/name registration fields from PHASE3-INTEGRATION.md Interface 1.
type telemetryRequest struct {
	AgentID   string `json:"agentId"`
	Framework string `json:"framework"`
	Action    string `json:"action"`
	Tool      string `json:"tool"`
	Target    string `json:"target"`
	Sensitive bool   `json:"sensitive"`
	Owner     string `json:"owner"`
	Name      string `json:"name"`
	Simulated bool   `json:"simulated"`
	Timestamp string `json:"timestamp"`
}

func (s *server) ingestTelemetry(c *gin.Context) {
	var req telemetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid body", err.Error())
		return
	}
	switch {
	case req.AgentID == "":
		httpx.Error(c, http.StatusBadRequest, "invalid body", "agentId is required")
		return
	case !validFrameworks[req.Framework]:
		httpx.Error(c, http.StatusBadRequest, "invalid body", "framework must be one of langgraph|crewai|autogen|openai-agents-sdk|mcp")
		return
	case !validActions[req.Action]:
		httpx.Error(c, http.StatusBadRequest, "invalid body", "action must be one of tool_call|data_access|privilege_escalation|external_call")
		return
	case !req.Simulated:
		// Brief R4: all agent telemetry on this path is SIMULATED, end to end.
		httpx.Error(c, http.StatusBadRequest, "invalid body", "simulated must be true (Brief R4)")
		return
	}
	occurred := time.Now().UTC()
	if req.Timestamp != "" {
		t, err := time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "invalid body", "timestamp must be RFC3339")
			return
		}
		occurred = t.UTC()
	}

	ctx := c.Request.Context()
	prev, score, factors, _, err := s.db.IngestAgentActivity(ctx,
		db.Agent{ID: req.AgentID, Name: req.Name, Framework: req.Framework, OwnerEmail: req.Owner},
		db.AgentActivity{Action: req.Action, Tool: req.Tool, Target: req.Target, Sensitive: req.Sensitive, OccurredAt: occurred},
		func(history []db.AgentActivity) (int, []string) {
			return agentrisk.Score(toRiskEvents(history))
		})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "persist failed", err.Error())
		return
	}

	// Publish ai.agent.activity (contracts/events/agent.activity.schema.json).
	evt := map[string]any{
		"eventId":   uuid.NewString(),
		"agentId":   req.AgentID,
		"framework": req.Framework,
		"action":    req.Action,
		"riskScore": score,
		"simulated": true,
		"timestamp": occurred.Format(time.RFC3339),
	}
	if req.Owner != "" {
		evt["owner"] = req.Owner
	}
	if req.Tool != "" {
		evt["tool"] = req.Tool
	}
	if req.Target != "" {
		evt["target"] = req.Target
	}
	if req.Action == agentrisk.ActionDataAccess {
		evt["sensitive"] = req.Sensitive
	}
	s.bus.Publish(bus.SubjectAgentActivity, evt)

	// Threshold semantics: emit exactly ONE violation on the <70 -> >=70 crossing.
	// prev comes from the locked agents row, so concurrent events serialize and
	// re-runs (score already >=70) never re-emit.
	if agentrisk.Crossed(prev, score) {
		s.emitThresholdViolation(ctx, req.AgentID, req.Owner, score, factors)
	}

	c.JSON(http.StatusAccepted, gin.H{"accepted": true, "agentId": req.AgentID, "riskScore": score})
}

// emitThresholdViolation maps category OTHER through the compliance fixtures
// (NIST AI RMF on the current fixture set — resolved at runtime, never
// hardcoded), inserts the violations row (prompt_id NULL, agent_id set) and
// publishes ai.violation.raised.
func (s *server) emitThresholdViolation(ctx context.Context, agentID, owner string, score int, factors []string) {
	matches := s.comp.MatchFrameworks([]string{"OTHER"})
	if len(matches) == 0 {
		log.Printf("[%s] no fixture framework maps OTHER — skipping violation for %s", serviceName, agentID)
		return
	}
	m := matches[0] // deterministic fixture load order; exactly one on the committed set
	desc := "Agent risk threshold breached (score " + strconv.Itoa(score) + "): " + strings.Join(factors, "; ")
	vid, err := s.db.InsertViolation(ctx, db.Violation{
		AgentID:     agentID,
		UserEmail:   owner,
		Framework:   m.Framework,
		ControlID:   m.Control.ControlID,
		Category:    m.Category,
		Severity:    m.Control.Severity,
		Description: desc,
	})
	if err != nil {
		log.Printf("[%s] insert threshold violation for %s: %v", serviceName, agentID, err)
		return
	}
	log.Printf("[%s] threshold breach: agent=%s score=%d violation=%s control=%s/%s",
		serviceName, agentID, score, vid, m.Framework, m.Control.ControlID)
	payload := map[string]any{
		"eventId":     uuid.NewString(),
		"violationId": vid,
		"agentId":     agentID,
		"framework":   m.Framework,
		"controlId":   m.Control.ControlID,
		"category":    m.Category,
		"severity":    m.Control.Severity,
		"description": desc,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	if owner != "" {
		payload["userEmail"] = owner
	}
	s.bus.Publish(bus.SubjectViolationRaised, payload)
}

// agentDTO matches contracts/openapi/agents.yaml Agent.
type agentDTO struct {
	AgentID     string        `json:"agentId"`
	Name        string        `json:"name,omitempty"`
	Framework   string        `json:"framework"`
	Owner       string        `json:"owner,omitempty"`
	Tools       []string      `json:"tools"`
	RiskScore   int           `json:"riskScore"`
	RiskFactors []string      `json:"riskFactors"`
	Simulated   bool          `json:"simulated"`
	LastSeen    time.Time     `json:"lastSeen"`
	Activity    []activityDTO `json:"activity,omitempty"` // detail view only
}

type activityDTO struct {
	Action     string    `json:"action"`
	Tool       string    `json:"tool,omitempty"`
	Target     string    `json:"target,omitempty"`
	Sensitive  bool      `json:"sensitive"`
	Simulated  bool      `json:"simulated"`
	OccurredAt time.Time `json:"occurredAt"`
}

func (s *server) listAgents(c *gin.Context) {
	ctx := c.Request.Context()
	agents, err := s.db.ListAgents(ctx)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	out := make([]agentDTO, 0, len(agents))
	for _, a := range agents {
		history, err := s.db.ListAgentActivity(ctx, a.ID, 0)
		if err != nil {
			httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
			return
		}
		out = append(out, toAgentDTO(a, history, false))
	}
	c.JSON(http.StatusOK, out)
}

func (s *server) getAgent(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	a, err := s.db.GetAgent(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(c, http.StatusNotFound, "not found", "unknown agent: "+id)
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	history, err := s.db.ListAgentActivity(ctx, id, 0)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, toAgentDTO(*a, history, true))
}

// toAgentDTO builds the API shape: tools = distinct tools from activity,
// riskFactors recomputed from the same history the score came from.
func toAgentDTO(a db.Agent, history []db.AgentActivity, withActivity bool) agentDTO {
	events := toRiskEvents(history)
	_, factors := agentrisk.Score(events)
	tools := []string{}
	seen := map[string]bool{}
	for _, e := range events {
		if e.Tool != "" && !seen[e.Tool] {
			seen[e.Tool] = true
			tools = append(tools, e.Tool)
		}
	}
	if factors == nil {
		factors = []string{}
	}
	dto := agentDTO{
		AgentID:     a.ID,
		Name:        a.Name,
		Framework:   a.Framework,
		Owner:       a.OwnerEmail,
		Tools:       tools,
		RiskScore:   a.RiskScore,
		RiskFactors: factors,
		Simulated:   a.Simulated,
		LastSeen:    a.LastSeen,
	}
	if withActivity {
		acts := make([]activityDTO, 0, len(history))
		for _, h := range history {
			acts = append(acts, activityDTO{
				Action: h.Action, Tool: h.Tool, Target: h.Target,
				Sensitive: h.Sensitive, Simulated: true, OccurredAt: h.OccurredAt,
			})
		}
		dto.Activity = acts
	}
	return dto
}

func toRiskEvents(history []db.AgentActivity) []agentrisk.Event {
	out := make([]agentrisk.Event, 0, len(history))
	for _, h := range history {
		out = append(out, agentrisk.Event{Action: h.Action, Tool: h.Tool, Target: h.Target, Sensitive: h.Sensitive})
	}
	return out
}
