package main

// Graph population: schema constraints, Postgres backfill, and live NATS
// subscription. Everything is MERGE-on-natural-key (email / provider / id /
// name — the same keys the db/neo4j/schema.cypher constraints enforce), so
// both the backfill and the live path are idempotent.

import (
	"context"
	"encoding/json"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/zealdefense/ai-sentinel/services/internal/bus"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
)

// schemaConstraints mirrors db/neo4j/schema.cypher (the committed Phase 0
// contract). IF NOT EXISTS makes every startup idempotent.
var schemaConstraints = []string{
	`CREATE CONSTRAINT user_email IF NOT EXISTS FOR (u:User) REQUIRE u.email IS UNIQUE`,
	`CREATE CONSTRAINT model_provider IF NOT EXISTS FOR (m:Model) REQUIRE m.provider IS UNIQUE`,
	`CREATE CONSTRAINT prompt_id IF NOT EXISTS FOR (p:Prompt) REQUIRE p.id IS UNIQUE`,
	`CREATE CONSTRAINT agent_id IF NOT EXISTS FOR (a:Agent) REQUIRE a.id IS UNIQUE`,
	`CREATE CONSTRAINT tool_name IF NOT EXISTS FOR (t:Tool) REQUIRE t.name IS UNIQUE`,
	`CREATE CONSTRAINT dataset_name IF NOT EXISTS FOR (d:Dataset) REQUIRE d.name IS UNIQUE`,
	`CREATE CONSTRAINT violation_id IF NOT EXISTS FOR (v:Violation) REQUIRE v.id IS UNIQUE`,
}

func applyConstraints(ctx context.Context, driver neo4j.DriverWithContext) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	for _, stmt := range schemaConstraints {
		if _, err := session.Run(ctx, stmt, nil); err != nil {
			return err
		}
	}
	return nil
}

// write runs one write query in its own managed transaction.
func write(ctx context.Context, driver neo4j.DriverWithContext, cypher string, params map[string]any) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err := neo4j.ExecuteWrite(ctx, session, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, cypher, params)
	})
	return err
}

// ---------------------------------------------------------------------------
// MERGE statements shared by backfill and the live NATS path
// ---------------------------------------------------------------------------

const mergeUser = `MERGE (u:User {email: $email})`

// mergeModel keeps an authoritative sanctioned flag when one is supplied
// (backfill reads it from the models table); the live discovery path passes
// onCreateOnly=true semantics via mergeModelDefault instead.
const mergeModel = `MERGE (m:Model {provider: $provider}) SET m.sanctioned = $sanctioned`

const mergeModelDefault = `MERGE (m:Model {provider: $provider}) ON CREATE SET m.sanctioned = false`

const mergePrompt = `
MERGE (u:User {email: $email})
MERGE (m:Model {provider: $provider}) ON CREATE SET m.sanctioned = false
MERGE (p:Prompt {id: $id})
SET p.decision = $decision, p.riskScore = $riskScore, p.categories = $categories, p.scenario = $scenario
MERGE (u)-[:GENERATED]->(p)
MERGE (u)-[:USED]->(m)
MERGE (p)-[:USED]->(m)
WITH p, $artifactName AS artifact
WHERE artifact <> ''
MERGE (d:Dataset {name: artifact})
MERGE (p)-[:EXPOSED]->(d)`

const mergeViolation = `
MERGE (v:Violation {id: $id})
SET v.framework = $framework, v.controlId = $controlId, v.severity = $severity, v.category = $category
WITH v, $promptId AS pid
WHERE pid <> ''
MERGE (p:Prompt {id: pid})
MERGE (p)-[:EXPOSED]->(v)`

const mergeAgent = `
MERGE (a:Agent {id: $id})
SET a.framework = $framework, a.riskScore = $riskScore
FOREACH (_ IN CASE WHEN $owner <> '' THEN [1] ELSE [] END | SET a.owner = $owner)
FOREACH (_ IN CASE WHEN $name <> '' THEN [1] ELSE [] END | SET a.name = $name)`

const mergeAgentTool = `
MERGE (a:Agent {id: $id})
MERGE (t:Tool {name: $tool})
MERGE (a)-[:USED]->(t)`

const mergeAgentDataset = `
MERGE (a:Agent {id: $id})
MERGE (d:Dataset {name: $dataset})
MERGE (a)-[:ACCESSED]->(d)`

const mergeUserModel = `
MERGE (u:User {email: $email})
MERGE (m:Model {provider: $provider}) ON CREATE SET m.sanctioned = false
MERGE (u)-[:USED]->(m)`

// ---------------------------------------------------------------------------
// Backfill from Postgres (source of truth)
// ---------------------------------------------------------------------------

func backfill(ctx context.Context, database *db.DB, driver neo4j.DriverWithContext) error {
	pool := database.Pool()
	counts := map[string]int{}

	// 1. Models (authoritative sanctioned flag).
	rows, err := pool.Query(ctx, `SELECT provider, sanctioned FROM models`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var provider string
		var sanctioned bool
		if err := rows.Scan(&provider, &sanctioned); err != nil {
			rows.Close()
			return err
		}
		if err := write(ctx, driver, mergeModel, map[string]any{"provider": provider, "sanctioned": sanctioned}); err != nil {
			rows.Close()
			return err
		}
		counts["models"]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 2. Users from prompts + discovery_events.
	rows, err = pool.Query(ctx,
		`SELECT DISTINCT user_email FROM prompts UNION SELECT DISTINCT user_email FROM discovery_events`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			rows.Close()
			return err
		}
		if err := write(ctx, driver, mergeUser, map[string]any{"email": email}); err != nil {
			rows.Close()
			return err
		}
		counts["users"]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 3. User -> Model edges observed via discovery.
	rows, err = pool.Query(ctx, `SELECT DISTINCT user_email, provider FROM discovery_events`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var email, provider string
		if err := rows.Scan(&email, &provider); err != nil {
			rows.Close()
			return err
		}
		if err := write(ctx, driver, mergeUserModel, map[string]any{"email": email, "provider": provider}); err != nil {
			rows.Close()
			return err
		}
		counts["user-model edges"]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 4. Prompts with aggregated finding categories (+ dataset from artifact).
	rows, err = pool.Query(ctx,
		`SELECT p.id::text, p.user_email, p.provider, p.decision, p.risk_score,
		        COALESCE(p.scenario,''), COALESCE(p.artifact_name,''),
		        COALESCE(ARRAY_REMOVE(ARRAY_AGG(DISTINCT f.category), NULL), '{}')
		 FROM prompts p LEFT JOIN findings f ON f.prompt_id = p.id
		 GROUP BY p.id`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, email, provider, decision, scenario, artifact string
		var riskScore int
		var categories []string
		if err := rows.Scan(&id, &email, &provider, &decision, &riskScore, &scenario, &artifact, &categories); err != nil {
			rows.Close()
			return err
		}
		if err := write(ctx, driver, mergePrompt, map[string]any{
			"id": id, "email": email, "provider": provider, "decision": decision,
			"riskScore": riskScore, "categories": categories, "scenario": scenario,
			"artifactName": artifact,
		}); err != nil {
			rows.Close()
			return err
		}
		counts["prompts"]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 5. Violations (linked to their prompt where present).
	rows, err = pool.Query(ctx,
		`SELECT id::text, COALESCE(prompt_id::text,''), framework, control_id, severity, category
		 FROM violations`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, promptID, framework, controlID, severity, category string
		if err := rows.Scan(&id, &promptID, &framework, &controlID, &severity, &category); err != nil {
			rows.Close()
			return err
		}
		if err := write(ctx, driver, mergeViolation, map[string]any{
			"id": id, "promptId": promptID, "framework": framework,
			"controlId": controlID, "severity": severity, "category": category,
		}); err != nil {
			rows.Close()
			return err
		}
		counts["violations"]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 6. Agents.
	rows, err = pool.Query(ctx,
		`SELECT id, COALESCE(name,''), framework, COALESCE(owner_email,''), risk_score FROM agents`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, name, framework, owner string
		var riskScore int
		if err := rows.Scan(&id, &name, &framework, &owner, &riskScore); err != nil {
			rows.Close()
			return err
		}
		if err := write(ctx, driver, mergeAgent, map[string]any{
			"id": id, "name": name, "framework": framework, "owner": owner, "riskScore": riskScore,
		}); err != nil {
			rows.Close()
			return err
		}
		counts["agents"]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 7. Agent -> Tool edges from activity.
	rows, err = pool.Query(ctx,
		`SELECT DISTINCT agent_id, tool FROM agent_activity WHERE tool IS NOT NULL AND tool <> ''`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var agentID, tool string
		if err := rows.Scan(&agentID, &tool); err != nil {
			rows.Close()
			return err
		}
		if err := write(ctx, driver, mergeAgentTool, map[string]any{"id": agentID, "tool": tool}); err != nil {
			rows.Close()
			return err
		}
		counts["agent-tool edges"]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 8. Agent -> Dataset edges from data_access activity.
	rows, err = pool.Query(ctx,
		`SELECT DISTINCT agent_id, target FROM agent_activity
		 WHERE action = 'data_access' AND target IS NOT NULL AND target <> ''`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var agentID, target string
		if err := rows.Scan(&agentID, &target); err != nil {
			rows.Close()
			return err
		}
		if err := write(ctx, driver, mergeAgentDataset, map[string]any{"id": agentID, "dataset": target}); err != nil {
			rows.Close()
			return err
		}
		counts["agent-dataset edges"]++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	log.Printf("[%s] backfill complete: %v", serviceName, counts)
	return nil
}

// ---------------------------------------------------------------------------
// Live NATS subscription (fail-soft)
// ---------------------------------------------------------------------------

func subscribeLive(ctx context.Context, b *bus.Bus, driver neo4j.DriverWithContext) {
	sub := func(subject string, handler func(map[string]any)) {
		_, err := b.Subscribe(subject, func(data []byte) {
			var evt map[string]any
			if err := json.Unmarshal(data, &evt); err != nil {
				log.Printf("[%s] bad %s payload: %v", serviceName, subject, err)
				return
			}
			handler(evt)
		})
		if err != nil {
			log.Printf("[%s] subscribe %s failed: %v — continuing (fail-soft)", serviceName, subject, err)
		}
	}

	sub(bus.SubjectPromptInspected, func(e map[string]any) {
		if str(e["promptId"]) == "" || str(e["userEmail"]) == "" || str(e["provider"]) == "" {
			return
		}
		cats := strSlice(e["categories"])
		err := write(ctx, driver, mergePrompt, map[string]any{
			"id":           str(e["promptId"]),
			"email":        str(e["userEmail"]),
			"provider":     str(e["provider"]),
			"decision":     str(e["decision"]),
			"riskScore":    num(e["riskScore"]),
			"categories":   cats,
			"scenario":     str(e["scenario"]),
			"artifactName": str(e["artifactName"]),
		})
		logMergeErr("ai.prompt.inspected", err)
	})

	sub(bus.SubjectViolationRaised, func(e map[string]any) {
		if str(e["violationId"]) == "" {
			return
		}
		err := write(ctx, driver, mergeViolation, map[string]any{
			"id":        str(e["violationId"]),
			"promptId":  str(e["promptId"]),
			"framework": str(e["framework"]),
			"controlId": str(e["controlId"]),
			"severity":  str(e["severity"]),
			"category":  str(e["category"]),
		})
		logMergeErr("ai.violation.raised", err)
	})

	sub(bus.SubjectAgentActivity, func(e map[string]any) {
		agentID := str(e["agentId"])
		if agentID == "" {
			return
		}
		err := write(ctx, driver, mergeAgent, map[string]any{
			"id": agentID, "framework": str(e["framework"]),
			"owner": str(e["owner"]), "name": "", "riskScore": num(e["riskScore"]),
		})
		logMergeErr("ai.agent.activity(agent)", err)
		if tool := str(e["tool"]); tool != "" {
			logMergeErr("ai.agent.activity(tool)",
				write(ctx, driver, mergeAgentTool, map[string]any{"id": agentID, "tool": tool}))
		}
		if str(e["action"]) == "data_access" {
			if target := str(e["target"]); target != "" {
				logMergeErr("ai.agent.activity(dataset)",
					write(ctx, driver, mergeAgentDataset, map[string]any{"id": agentID, "dataset": target}))
			}
		}
	})

	sub(bus.SubjectDiscoveryObserved, func(e map[string]any) {
		email, provider := str(e["userEmail"]), str(e["provider"])
		if email == "" || provider == "" {
			return
		}
		logMergeErr("ai.discovery.observed",
			write(ctx, driver, mergeUserModel, map[string]any{"email": email, "provider": provider}))
	})
}

func logMergeErr(what string, err error) {
	if err != nil {
		log.Printf("[%s] live merge %s failed: %v", serviceName, what, err)
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func strSlice(v any) []string {
	arr, _ := v.([]any)
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
