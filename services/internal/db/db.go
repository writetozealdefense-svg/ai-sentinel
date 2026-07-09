// Package db wraps a pgx/v5 pool and provides the insert/query helpers the three
// services need against the committed schema (db/postgres/001_init.sql).
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is a pgx pool wrapper.
type DB struct {
	pool *pgxpool.Pool
}

// Connect opens a pgx pool against databaseURL and pings it.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &DB{pool: pool}, nil
}

// Close releases the pool.
func (d *DB) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}

// Pool exposes the underlying pool for ad-hoc queries.
func (d *DB) Pool() *pgxpool.Pool { return d.pool }

// ---------------------------------------------------------------------------
// models
// ---------------------------------------------------------------------------

// EnsureModel inserts (provider, sanctioned default false) if absent and returns
// the current sanctioned flag for the provider.
func (d *DB) EnsureModel(ctx context.Context, provider string) (sanctioned bool, err error) {
	_, err = d.pool.Exec(ctx,
		`INSERT INTO models (provider, sanctioned) VALUES ($1, FALSE)
		 ON CONFLICT (provider) DO NOTHING`, provider)
	if err != nil {
		return false, fmt.Errorf("ensure model: %w", err)
	}
	err = d.pool.QueryRow(ctx, `SELECT sanctioned FROM models WHERE provider = $1`, provider).Scan(&sanctioned)
	if err != nil {
		return false, fmt.Errorf("select model sanctioned: %w", err)
	}
	return sanctioned, nil
}

// Sanctioned returns the sanctioned flag for a provider, defaulting to false if absent.
func (d *DB) Sanctioned(ctx context.Context, provider string) (bool, error) {
	var s bool
	err := d.pool.QueryRow(ctx, `SELECT COALESCE(sanctioned, FALSE) FROM models WHERE provider = $1`, provider).Scan(&s)
	if err != nil {
		// Treat "no rows" as unsanctioned without surfacing an error.
		return false, nil
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// discovery_events
// ---------------------------------------------------------------------------

// DiscoveryEvent is a row in discovery_events.
type DiscoveryEvent struct {
	ID           string
	Provider     string
	UserEmail    string
	Source       string
	Action       string
	ArtifactName string
	Bytes        int64
	Simulated    bool
	OccurredAt   time.Time
}

// InsertDiscoveryEvent inserts a discovery_events row and returns its id.
func (d *DB) InsertDiscoveryEvent(ctx context.Context, e DiscoveryEvent) (string, error) {
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	var id string
	err := d.pool.QueryRow(ctx,
		`INSERT INTO discovery_events
		   (provider, user_email, source, action, artifact_name, bytes, simulated, occurred_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id`,
		e.Provider, e.UserEmail, e.Source, nullStr(e.Action), nullStr(e.ArtifactName),
		e.Bytes, e.Simulated, e.OccurredAt).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert discovery_event: %w", err)
	}
	return id, nil
}

// ListDiscoveryEvents returns recent events, optionally filtered by provider.
func (d *DB) ListDiscoveryEvents(ctx context.Context, provider string, limit int) ([]DiscoveryEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	q := `SELECT id, provider, user_email, source, COALESCE(action,''), COALESCE(artifact_name,''),
	             COALESCE(bytes,0), simulated, occurred_at
	      FROM discovery_events`
	args := []any{}
	if provider != "" {
		q += ` WHERE provider = $1`
		args = append(args, provider)
		q += ` ORDER BY occurred_at DESC LIMIT $2`
		args = append(args, limit)
	} else {
		q += ` ORDER BY occurred_at DESC LIMIT $1`
		args = append(args, limit)
	}
	rows, err := d.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list discovery_events: %w", err)
	}
	defer rows.Close()
	var out []DiscoveryEvent
	for rows.Next() {
		var e DiscoveryEvent
		if err := rows.Scan(&e.ID, &e.Provider, &e.UserEmail, &e.Source, &e.Action,
			&e.ArtifactName, &e.Bytes, &e.Simulated, &e.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan discovery_event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// InventoryItem aggregates discovery_events by provider.
type InventoryItem struct {
	Provider    string
	UsageCount  int
	UniqueUsers int
	Sanctioned  bool
	FirstSeen   time.Time
	LastSeen    time.Time
}

// Inventory aggregates discovery_events by provider, joining the models table for
// the sanctioned flag.
func (d *DB) Inventory(ctx context.Context) ([]InventoryItem, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT de.provider,
		        COUNT(*)                         AS usage_count,
		        COUNT(DISTINCT de.user_email)    AS unique_users,
		        COALESCE(m.sanctioned, FALSE)    AS sanctioned,
		        MIN(de.occurred_at)              AS first_seen,
		        MAX(de.occurred_at)              AS last_seen
		 FROM discovery_events de
		 LEFT JOIN models m ON m.provider = de.provider
		 GROUP BY de.provider, m.sanctioned
		 ORDER BY usage_count DESC`)
	if err != nil {
		return nil, fmt.Errorf("inventory query: %w", err)
	}
	defer rows.Close()
	var out []InventoryItem
	for rows.Next() {
		var it InventoryItem
		if err := rows.Scan(&it.Provider, &it.UsageCount, &it.UniqueUsers, &it.Sanctioned,
			&it.FirstSeen, &it.LastSeen); err != nil {
			return nil, fmt.Errorf("scan inventory: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// prompts + findings
// ---------------------------------------------------------------------------

// Prompt is a row in prompts.
type Prompt struct {
	ID              string
	UserEmail       string
	Provider        string
	ArtifactName    string
	ContentType     string
	Decision        string
	RiskScore       int
	RedactedPreview string
	PolicyTrace     string
	Scenario        string
	OccurredAt      time.Time
}

// Finding is a row in findings (aggregated per type).
type Finding struct {
	ID          string
	PromptID    string
	Type        string
	Category    string
	Engine      string
	Score       float64
	RuleName    string
	Occurrences int
}

// InsertPrompt inserts a prompts row and returns its id.
func (d *DB) InsertPrompt(ctx context.Context, p Prompt) (string, error) {
	if p.ContentType == "" {
		p.ContentType = "text"
	}
	var id string
	err := d.pool.QueryRow(ctx,
		`INSERT INTO prompts
		   (user_email, provider, artifact_name, content_type, decision, risk_score,
		    redacted_preview, policy_trace, scenario)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id`,
		p.UserEmail, p.Provider, nullStr(p.ArtifactName), p.ContentType, p.Decision,
		p.RiskScore, nullStr(p.RedactedPreview), nullStr(p.PolicyTrace), nullStr(p.Scenario)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert prompt: %w", err)
	}
	return id, nil
}

// InsertFindings bulk-inserts findings for a prompt.
func (d *DB) InsertFindings(ctx context.Context, promptID string, fs []Finding) error {
	for _, f := range fs {
		_, err := d.pool.Exec(ctx,
			`INSERT INTO findings (prompt_id, type, category, engine, score, rule_name, occurrences)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			promptID, f.Type, f.Category, f.Engine, f.Score, nullStr(f.RuleName), f.Occurrences)
		if err != nil {
			return fmt.Errorf("insert finding %s: %w", f.Type, err)
		}
	}
	return nil
}

// PromptSummary is a list-view row joining a prompt with the distinct set of its
// finding categories (contracts/openapi/prompts.yaml PromptSummary).
type PromptSummary struct {
	ID           string
	UserEmail    string
	Provider     string
	ArtifactName string
	Decision     string
	RiskScore    int
	Categories   []string
	Scenario     string
	OccurredAt   time.Time
}

// ListPrompts returns recent prompts newest-first, optionally filtered by decision,
// each with the distinct set of its findings' categories. limit defaults to 50 and
// is capped at 500.
func (d *DB) ListPrompts(ctx context.Context, decision string, limit int) ([]PromptSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	// Aggregate distinct finding categories per prompt via a left join so prompts
	// with no findings still appear (empty category set).
	q := `SELECT p.id, p.user_email, p.provider, COALESCE(p.artifact_name,''),
	             p.decision, p.risk_score, COALESCE(p.scenario,''), p.occurred_at,
	             COALESCE(ARRAY_REMOVE(ARRAY_AGG(DISTINCT f.category), NULL), '{}') AS categories
	      FROM prompts p
	      LEFT JOIN findings f ON f.prompt_id = p.id`
	args := []any{}
	if decision != "" {
		q += ` WHERE p.decision = $1`
		args = append(args, decision)
	}
	q += ` GROUP BY p.id`
	if decision != "" {
		q += ` ORDER BY p.occurred_at DESC LIMIT $2`
	} else {
		q += ` ORDER BY p.occurred_at DESC LIMIT $1`
	}
	args = append(args, limit)

	rows, err := d.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list prompts: %w", err)
	}
	defer rows.Close()
	var out []PromptSummary
	for rows.Next() {
		var s PromptSummary
		if err := rows.Scan(&s.ID, &s.UserEmail, &s.Provider, &s.ArtifactName,
			&s.Decision, &s.RiskScore, &s.Scenario, &s.OccurredAt, &s.Categories); err != nil {
			return nil, fmt.Errorf("scan prompt summary: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetPrompt loads a prompt row and its findings by id.
func (d *DB) GetPrompt(ctx context.Context, id string) (*Prompt, []Finding, error) {
	var p Prompt
	err := d.pool.QueryRow(ctx,
		`SELECT id, user_email, provider, COALESCE(artifact_name,''), content_type, decision,
		        risk_score, COALESCE(redacted_preview,''), COALESCE(policy_trace,''),
		        COALESCE(scenario,''), occurred_at
		 FROM prompts WHERE id = $1`, id).Scan(
		&p.ID, &p.UserEmail, &p.Provider, &p.ArtifactName, &p.ContentType, &p.Decision,
		&p.RiskScore, &p.RedactedPreview, &p.PolicyTrace, &p.Scenario, &p.OccurredAt)
	if err != nil {
		return nil, nil, err
	}
	rows, err := d.pool.Query(ctx,
		`SELECT id, prompt_id, type, category, engine, score, COALESCE(rule_name,''), occurrences
		 FROM findings WHERE prompt_id = $1 ORDER BY type`, id)
	if err != nil {
		return nil, nil, fmt.Errorf("load findings: %w", err)
	}
	defer rows.Close()
	var fs []Finding
	for rows.Next() {
		var f Finding
		if err := rows.Scan(&f.ID, &f.PromptID, &f.Type, &f.Category, &f.Engine,
			&f.Score, &f.RuleName, &f.Occurrences); err != nil {
			return nil, nil, fmt.Errorf("scan finding: %w", err)
		}
		fs = append(fs, f)
	}
	return &p, fs, rows.Err()
}

// ---------------------------------------------------------------------------
// violations
// ---------------------------------------------------------------------------

// Violation is a row in violations.
type Violation struct {
	ID          string
	PromptID    string
	AgentID     string
	UserEmail   string
	Framework   string
	ControlID   string
	Category    string
	Severity    string
	Description string
	OccurredAt  time.Time
}

// InsertViolation inserts a violations row and returns its id. PromptID and
// AgentID are both optional (empty -> NULL); agent-svc sets AgentID with a NULL
// prompt_id, prompt-svc the reverse.
func (d *DB) InsertViolation(ctx context.Context, v Violation) (string, error) {
	var id string
	err := d.pool.QueryRow(ctx,
		`INSERT INTO violations
		   (prompt_id, agent_id, user_email, framework, control_id, category, severity, description)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id`,
		nullStr(v.PromptID), nullStr(v.AgentID), nullStr(v.UserEmail), v.Framework, v.ControlID,
		v.Category, v.Severity, nullStr(v.Description)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert violation: %w", err)
	}
	return id, nil
}

// ListViolations returns violations, optionally filtered by framework.
func (d *DB) ListViolations(ctx context.Context, framework string, limit int) ([]Violation, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, COALESCE(prompt_id::text,''), COALESCE(user_email,''), framework,
	             control_id, category, severity, COALESCE(description,''), occurred_at
	      FROM violations`
	args := []any{}
	if framework != "" {
		q += ` WHERE framework = $1 ORDER BY occurred_at DESC LIMIT $2`
		args = append(args, framework, limit)
	} else {
		q += ` ORDER BY occurred_at DESC LIMIT $1`
		args = append(args, limit)
	}
	rows, err := d.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list violations: %w", err)
	}
	defer rows.Close()
	var out []Violation
	for rows.Next() {
		var v Violation
		if err := rows.Scan(&v.ID, &v.PromptID, &v.UserEmail, &v.Framework, &v.ControlID,
			&v.Category, &v.Severity, &v.Description, &v.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan violation: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// CountOpenViolations returns the number of violation rows for a framework.
func (d *DB) CountOpenViolations(ctx context.Context, framework string) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, `SELECT COUNT(*) FROM violations WHERE framework = $1`, framework).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count violations: %w", err)
	}
	return n, nil
}

// nullStr converts an empty string to a SQL NULL via *string.
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
