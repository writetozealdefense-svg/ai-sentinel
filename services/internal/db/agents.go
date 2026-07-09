package db

// Agent + agent_activity helpers (Phase 3, agent-svc). Tables from
// db/postgres/001_init.sql: agents, agent_activity.

import (
	"context"
	"fmt"
	"time"
)

// Agent is a row in agents.
type Agent struct {
	ID         string
	Name       string
	Framework  string
	OwnerEmail string
	RiskScore  int
	Simulated  bool
	LastSeen   time.Time
}

// AgentActivity is a row in agent_activity.
type AgentActivity struct {
	ID         string
	AgentID    string
	Action     string
	Tool       string
	Target     string
	Sensitive  bool
	OccurredAt time.Time
}

// IngestAgentActivity atomically: upserts the agents row (locking it for the
// duration of the transaction so threshold-crossing detection is serialized),
// inserts the activity row, loads the agent's full history, recomputes the risk
// score via compute, and persists the new score. It returns the previous score,
// the new score, the risk factors, and the full history.
//
// Name/owner are only overwritten when non-empty so later events without them
// don't erase the registration data. simulated is pinned TRUE (Brief R4).
func (d *DB) IngestAgentActivity(
	ctx context.Context,
	a Agent,
	act AgentActivity,
	compute func(history []AgentActivity) (score int, factors []string),
) (prevScore, newScore int, factors []string, history []AgentActivity, err error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Upsert + row lock. risk_score is untouched here, so RETURNING yields the
	// score as it stood BEFORE this event.
	err = tx.QueryRow(ctx,
		`INSERT INTO agents (id, name, framework, owner_email, simulated, last_seen)
		 VALUES ($1, NULLIF($2,''), $3, NULLIF($4,''), TRUE, $5)
		 ON CONFLICT (id) DO UPDATE SET
		   name        = COALESCE(NULLIF(EXCLUDED.name,''), agents.name),
		   framework   = EXCLUDED.framework,
		   owner_email = COALESCE(NULLIF(EXCLUDED.owner_email,''), agents.owner_email),
		   simulated   = TRUE,
		   last_seen   = EXCLUDED.last_seen
		 RETURNING risk_score`,
		a.ID, a.Name, a.Framework, a.OwnerEmail, act.OccurredAt).Scan(&prevScore)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("upsert agent: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO agent_activity (agent_id, action, tool, target, sensitive, occurred_at)
		 VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6)`,
		a.ID, act.Action, act.Tool, act.Target, act.Sensitive, act.OccurredAt)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("insert agent_activity: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT id, agent_id, action, COALESCE(tool,''), COALESCE(target,''), sensitive, occurred_at
		 FROM agent_activity WHERE agent_id = $1 ORDER BY occurred_at, id`, a.ID)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("load agent history: %w", err)
	}
	for rows.Next() {
		var h AgentActivity
		if err := rows.Scan(&h.ID, &h.AgentID, &h.Action, &h.Tool, &h.Target, &h.Sensitive, &h.OccurredAt); err != nil {
			rows.Close()
			return 0, 0, nil, nil, fmt.Errorf("scan agent history: %w", err)
		}
		history = append(history, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, nil, nil, fmt.Errorf("iterate agent history: %w", err)
	}

	newScore, factors = compute(history)

	if _, err = tx.Exec(ctx, `UPDATE agents SET risk_score = $2 WHERE id = $1`, a.ID, newScore); err != nil {
		return 0, 0, nil, nil, fmt.Errorf("update agent risk_score: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, 0, nil, nil, fmt.Errorf("commit: %w", err)
	}
	return prevScore, newScore, factors, history, nil
}

// ListAgents returns all agents ordered by risk score (desc), newest first on ties.
func (d *DB) ListAgents(ctx context.Context) ([]Agent, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT id, COALESCE(name,''), framework, COALESCE(owner_email,''), risk_score, simulated,
		        COALESCE(last_seen, 'epoch'::timestamptz)
		 FROM agents ORDER BY risk_score DESC, last_seen DESC`)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()
	var out []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.Name, &a.Framework, &a.OwnerEmail, &a.RiskScore, &a.Simulated, &a.LastSeen); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAgent loads one agent by id (pgx.ErrNoRows if absent).
func (d *DB) GetAgent(ctx context.Context, id string) (*Agent, error) {
	var a Agent
	err := d.pool.QueryRow(ctx,
		`SELECT id, COALESCE(name,''), framework, COALESCE(owner_email,''), risk_score, simulated,
		        COALESCE(last_seen, 'epoch'::timestamptz)
		 FROM agents WHERE id = $1`, id).Scan(
		&a.ID, &a.Name, &a.Framework, &a.OwnerEmail, &a.RiskScore, &a.Simulated, &a.LastSeen)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAgentActivity returns an agent's activity, oldest first (deterministic
// input order for the risk formula), capped at limit (default 200).
func (d *DB) ListAgentActivity(ctx context.Context, agentID string, limit int) ([]AgentActivity, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := d.pool.Query(ctx,
		`SELECT id, agent_id, action, COALESCE(tool,''), COALESCE(target,''), sensitive, occurred_at
		 FROM agent_activity WHERE agent_id = $1 ORDER BY occurred_at, id LIMIT $2`, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list agent_activity: %w", err)
	}
	defer rows.Close()
	var out []AgentActivity
	for rows.Next() {
		var h AgentActivity
		if err := rows.Scan(&h.ID, &h.AgentID, &h.Action, &h.Tool, &h.Target, &h.Sensitive, &h.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan agent_activity: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
