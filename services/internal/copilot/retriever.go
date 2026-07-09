package copilot

// Evidence-bundle retrieval (RAG over the platform's OWN data). All SQL here
// reads the committed schema db/postgres/001_init.sql via the shared pgx pool;
// control text is resolved through internal/compliance, never hardcoded.

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/zealdefense/ai-sentinel/services/internal/agentrisk"
	"github.com/zealdefense/ai-sentinel/services/internal/compliance"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
)

const aggregateCitationCap = 3 // recent violation rows cited per aggregate answer

// Retriever assembles evidence bundles and the router's aggregates.
type Retriever struct {
	DB   *db.DB
	Comp *compliance.Set
}

// EvidenceBundle retrieves the full bundle for a violation id:
// the violations row; the linked prompts row + aggregated findings + the
// provider's sanctioned flag (prompt violations); the agents row + recomputed
// risk factors (agent violations); and the matching fixture control.
// Returns pgx.ErrNoRows when the violation id is unknown.
func (r *Retriever) EvidenceBundle(ctx context.Context, violationID string) (*Bundle, error) {
	v, err := r.getViolation(ctx, violationID)
	if err != nil {
		return nil, err
	}
	b := &Bundle{Violation: *v, Control: FindControl(r.Comp, v.Framework, v.ControlID)}

	if v.PromptID != "" {
		p, fs, err := r.DB.GetPrompt(ctx, v.PromptID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// prompt row deleted (ON DELETE SET NULL race) — bundle degrades to
			// the violation row alone rather than failing.
		case err != nil:
			return nil, fmt.Errorf("load prompt %s: %w", v.PromptID, err)
		default:
			b.Prompt = p
			b.Findings = fs
			sanctioned, err := r.DB.Sanctioned(ctx, p.Provider)
			if err != nil {
				return nil, fmt.Errorf("load sanctioned flag for %s: %w", p.Provider, err)
			}
			b.Sanctioned = sanctioned
		}
	} else if v.AgentID != "" {
		a, err := r.DB.GetAgent(ctx, v.AgentID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// agent row deleted — degrade to the violation row alone.
		case err != nil:
			return nil, fmt.Errorf("load agent %s: %w", v.AgentID, err)
		default:
			b.Agent = a
			history, err := r.DB.ListAgentActivity(ctx, a.ID, 0)
			if err != nil {
				return nil, fmt.Errorf("load agent activity %s: %w", a.ID, err)
			}
			events := make([]agentrisk.Event, 0, len(history))
			for _, h := range history {
				events = append(events, agentrisk.Event{Action: h.Action, Tool: h.Tool, Target: h.Target, Sensitive: h.Sensitive})
			}
			_, factors := agentrisk.Score(events)
			b.AgentRiskFactors = factors
		}
	}
	return b, nil
}

// getViolation loads one violations row by id (pgx.ErrNoRows if absent).
// internal/db has no by-id violation getter, so the query lives here — the
// copilot is the only consumer that needs agent_id on the row.
func (r *Retriever) getViolation(ctx context.Context, id string) (*db.Violation, error) {
	var v db.Violation
	err := r.DB.Pool().QueryRow(ctx,
		`SELECT id, COALESCE(prompt_id::text,''), COALESCE(agent_id,''), COALESCE(user_email,''),
		        framework, control_id, category, severity, COALESCE(description,''), occurred_at
		 FROM violations WHERE id = $1`, id).Scan(
		&v.ID, &v.PromptID, &v.AgentID, &v.UserEmail,
		&v.Framework, &v.ControlID, &v.Category, &v.Severity, &v.Description, &v.OccurredAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// MostRecentBlockViolationID returns the newest violation whose linked prompt
// decision is BLOCK, or "" when none exists.
func (r *Retriever) MostRecentBlockViolationID(ctx context.Context) (string, error) {
	var id string
	err := r.DB.Pool().QueryRow(ctx,
		`SELECT v.id FROM violations v
		 JOIN prompts p ON p.id = v.prompt_id
		 WHERE p.decision = 'BLOCK'
		 ORDER BY v.occurred_at DESC, v.id LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("most recent BLOCK violation: %w", err)
	}
	return id, nil
}

// RiskiestUser returns the user with the most recorded violations plus their
// most recent violation rows, or nil when no violation carries a user.
func (r *Retriever) RiskiestUser(ctx context.Context) (*UserRiskSummary, error) {
	var u UserRiskSummary
	err := r.DB.Pool().QueryRow(ctx,
		`SELECT user_email, COUNT(*) FROM violations
		 WHERE COALESCE(user_email,'') <> ''
		 GROUP BY user_email
		 ORDER BY COUNT(*) DESC, user_email LIMIT 1`).Scan(&u.Email, &u.ViolationCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("riskiest user: %w", err)
	}
	recent, err := r.recentViolations(ctx, `WHERE user_email = $1`, u.Email)
	if err != nil {
		return nil, err
	}
	u.Recent = recent
	return &u, nil
}

// RiskiestAgent returns the agent with the top risk_score (ties broken by
// last_seen, matching internal/db ListAgents ordering) plus its recomputed
// risk factors and violation rows, or nil when no agents exist.
func (r *Retriever) RiskiestAgent(ctx context.Context) (*AgentRiskSummary, error) {
	agents, err := r.DB.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("riskiest agent: %w", err)
	}
	if len(agents) == 0 {
		return nil, nil
	}
	a := agents[0]
	history, err := r.DB.ListAgentActivity(ctx, a.ID, 0)
	if err != nil {
		return nil, fmt.Errorf("riskiest agent activity: %w", err)
	}
	events := make([]agentrisk.Event, 0, len(history))
	for _, h := range history {
		events = append(events, agentrisk.Event{Action: h.Action, Tool: h.Tool, Target: h.Target, Sensitive: h.Sensitive})
	}
	_, factors := agentrisk.Score(events)
	vs, err := r.recentViolations(ctx, `WHERE agent_id = $1`, a.ID)
	if err != nil {
		return nil, err
	}
	return &AgentRiskSummary{Agent: a, Factors: factors, Violations: vs}, nil
}

// CompliancePosture returns per-framework open-violation counts plus the most
// recent violation per framework, in deterministic framework order.
func (r *Retriever) CompliancePosture(ctx context.Context) ([]FrameworkPosture, error) {
	counts := map[string]int{}
	var order []string
	rows, err := r.DB.Pool().Query(ctx,
		`SELECT framework, COUNT(*) FROM violations GROUP BY framework ORDER BY framework`)
	if err != nil {
		return nil, fmt.Errorf("posture counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var fw string
		var n int
		if err := rows.Scan(&fw, &n); err != nil {
			return nil, fmt.Errorf("scan posture count: %w", err)
		}
		counts[fw] = n
		order = append(order, fw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	latest := map[string]db.Violation{}
	lrows, err := r.DB.Pool().Query(ctx,
		`SELECT DISTINCT ON (framework)
		        id, COALESCE(prompt_id::text,''), COALESCE(agent_id,''), COALESCE(user_email,''),
		        framework, control_id, category, severity, COALESCE(description,''), occurred_at
		 FROM violations ORDER BY framework, occurred_at DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("posture latest: %w", err)
	}
	defer lrows.Close()
	for lrows.Next() {
		var v db.Violation
		if err := lrows.Scan(&v.ID, &v.PromptID, &v.AgentID, &v.UserEmail,
			&v.Framework, &v.ControlID, &v.Category, &v.Severity, &v.Description, &v.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan posture latest: %w", err)
		}
		latest[v.Framework] = v
	}
	if err := lrows.Err(); err != nil {
		return nil, err
	}

	out := make([]FrameworkPosture, 0, len(order))
	for _, fw := range order {
		p := FrameworkPosture{Framework: fw, OpenViolations: counts[fw]}
		if v, ok := latest[fw]; ok {
			vv := v
			p.Latest = &vv
		}
		out = append(out, p)
	}
	return out, nil
}

// recentViolations loads the newest violation rows matching a WHERE clause
// with one $1 argument, capped at aggregateCitationCap.
func (r *Retriever) recentViolations(ctx context.Context, where string, arg any) ([]db.Violation, error) {
	rows, err := r.DB.Pool().Query(ctx,
		`SELECT id, COALESCE(prompt_id::text,''), COALESCE(agent_id,''), COALESCE(user_email,''),
		        framework, control_id, category, severity, COALESCE(description,''), occurred_at
		 FROM violations `+where+`
		 ORDER BY occurred_at DESC, id LIMIT `+fmt.Sprint(aggregateCitationCap), arg)
	if err != nil {
		return nil, fmt.Errorf("recent violations: %w", err)
	}
	defer rows.Close()
	var out []db.Violation
	for rows.Next() {
		var v db.Violation
		if err := rows.Scan(&v.ID, &v.PromptID, &v.AgentID, &v.UserEmail,
			&v.Framework, &v.ControlID, &v.Category, &v.Severity, &v.Description, &v.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan recent violation: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
