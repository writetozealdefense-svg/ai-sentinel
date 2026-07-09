package main

// The named queries (PHASE3-INTEGRATION.md Interface 2). Only these run — no
// arbitrary Cypher crosses the wire. Each query has a tabular `Cypher` (the
// exact string returned in the response for UI authenticity) and a companion
// `graphCypher` built from the same MATCH that returns the nodes/relationships
// so the response also carries a renderable subgraph.

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

type graphNode struct {
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	Properties map[string]any `json:"properties"`
}

type graphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type graphResult struct {
	Rows   []map[string]any `json:"rows"`
	Nodes  []graphNode      `json:"nodes"`
	Edges  []graphEdge      `json:"edges"`
	Cypher string           `json:"cypher"`
}

type namedQuery struct {
	Name        string
	Description string
	Cypher      string // tabular rows query — returned verbatim in the response
	graphCypher string // companion subgraph query (same MATCH, returns nodes/rels)
}

var namedQueryOrder = []string{
	"phi-to-unauthorized-ai",
	"high-risk-agents",
	"blocked-prompts-by-user",
	"full-graph",
}

var namedQueries = map[string]namedQuery{
	"phi-to-unauthorized-ai": {
		Name:        "phi-to-unauthorized-ai",
		Description: "All users who uploaded PHI to unauthorized (unsanctioned) AI systems, with the exposed dataset.",
		// Exact Cypher from PHASE3-INTEGRATION.md Interface 2.
		Cypher: `MATCH (u:User)-[:GENERATED]->(p:Prompt)-[:USED]->(m:Model {sanctioned:false})
WHERE 'PHI' IN p.categories
OPTIONAL MATCH (p)-[:EXPOSED]->(d:Dataset)
RETURN u.email AS user, m.provider AS aiSystem, d.name AS dataset,
       p.riskScore AS riskScore, p.decision AS decision`,
		graphCypher: `MATCH (u:User)-[g:GENERATED]->(p:Prompt)-[us:USED]->(m:Model {sanctioned:false})
WHERE 'PHI' IN p.categories
OPTIONAL MATCH (p)-[e:EXPOSED]->(d:Dataset)
RETURN u, g, p, us, m, e, d`,
	},
	"high-risk-agents": {
		Name:        "high-risk-agents",
		Description: "Agents with riskScore >= 70 and the tools they use.",
		Cypher: `MATCH (a:Agent)
WHERE a.riskScore >= 70
OPTIONAL MATCH (a)-[:USED]->(t:Tool)
RETURN a.id AS agent, a.framework AS framework, a.owner AS owner,
       a.riskScore AS riskScore, collect(DISTINCT t.name) AS tools`,
		graphCypher: `MATCH (a:Agent)
WHERE a.riskScore >= 70
OPTIONAL MATCH (a)-[u:USED]->(t:Tool)
OPTIONAL MATCH (a)-[ac:ACCESSED]->(d:Dataset)
RETURN a, u, t, ac, d`,
	},
	"blocked-prompts-by-user": {
		Name:        "blocked-prompts-by-user",
		Description: "Users, their blocked prompts, and the models those prompts targeted (overview subgraph).",
		Cypher: `MATCH (u:User)-[:GENERATED]->(p:Prompt)-[:USED]->(m:Model)
WHERE p.decision = 'BLOCK'
RETURN u.email AS user, p.id AS promptId, p.riskScore AS riskScore,
       p.categories AS categories, m.provider AS model
ORDER BY u.email`,
		graphCypher: `MATCH (u:User)-[g:GENERATED]->(p:Prompt)-[us:USED]->(m:Model)
WHERE p.decision = 'BLOCK'
RETURN u, g, p, us, m`,
	},
	"full-graph": {
		Name:        "full-graph",
		Description: "Bounded whole-graph view (~150 nodes) for the animated UI.",
		Cypher: `MATCH (n)
WITH labels(n)[0] AS label, count(n) AS count
RETURN label, count
ORDER BY count DESC`,
		graphCypher: `MATCH (n)
WITH n LIMIT 150
OPTIONAL MATCH (n)-[r]->(m)
RETURN n, r, m`,
	},
}

// execute runs the tabular query then the companion subgraph query and merges
// them into the graph.yaml GraphResult shape.
func (q namedQuery) execute(ctx context.Context, driver neo4j.DriverWithContext) (*graphResult, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	rows, err := neo4j.ExecuteRead(ctx, session, func(tx neo4j.ManagedTransaction) ([]map[string]any, error) {
		res, err := tx.Run(ctx, q.Cypher, nil)
		if err != nil {
			return nil, err
		}
		var out []map[string]any
		for res.Next(ctx) {
			rec := res.Record()
			row := map[string]any{}
			for i, key := range rec.Keys {
				row[key] = jsonValue(rec.Values[i])
			}
			out = append(out, row)
		}
		return out, res.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("rows query: %w", err)
	}

	nodes, edges, err := runSubgraph(ctx, session, q.graphCypher)
	if err != nil {
		return nil, fmt.Errorf("subgraph query: %w", err)
	}

	if rows == nil {
		rows = []map[string]any{}
	}
	return &graphResult{Rows: rows, Nodes: nodes, Edges: edges, Cypher: q.Cypher}, nil
}

// runSubgraph executes the companion query and folds every Node/Relationship/
// Path value in the result into a deduplicated {nodes, edges} subgraph with
// stable string ids (Label:naturalKey, e.g. "User:nurse.joy@example.com").
func runSubgraph(ctx context.Context, session neo4j.SessionWithContext, cypher string) ([]graphNode, []graphEdge, error) {
	type rel struct{ startEid, endEid, typ string }

	nodesByID := map[string]graphNode{}
	var nodeOrder []string
	eidToStable := map[string]string{}
	var rels []rel

	addNode := func(n dbtype.Node) {
		id := stableNodeID(n)
		eidToStable[n.ElementId] = id
		if _, seen := nodesByID[id]; !seen {
			label := ""
			if len(n.Labels) > 0 {
				label = n.Labels[0]
			}
			props := map[string]any{}
			for k, v := range n.Props {
				props[k] = jsonValue(v)
			}
			nodesByID[id] = graphNode{ID: id, Label: label, Properties: props}
			nodeOrder = append(nodeOrder, id)
		}
	}
	addRel := func(r dbtype.Relationship) {
		rels = append(rels, rel{startEid: r.StartElementId, endEid: r.EndElementId, typ: r.Type})
	}
	var addValue func(v any)
	addValue = func(v any) {
		switch t := v.(type) {
		case dbtype.Node:
			addNode(t)
		case dbtype.Relationship:
			addRel(t)
		case dbtype.Path:
			for _, n := range t.Nodes {
				addNode(n)
			}
			for _, r := range t.Relationships {
				addRel(r)
			}
		case []any:
			for _, x := range t {
				addValue(x)
			}
		}
	}

	// The callback must return a non-nil value: the driver's ExecuteRead does
	// result.(T), which panics on a nil interface (neo4j-go-driver v5).
	_, err := neo4j.ExecuteRead(ctx, session, func(tx neo4j.ManagedTransaction) (struct{}, error) {
		res, err := tx.Run(ctx, cypher, nil)
		if err != nil {
			return struct{}{}, err
		}
		for res.Next(ctx) {
			for _, v := range res.Record().Values {
				addValue(v)
			}
		}
		return struct{}{}, res.Err()
	})
	if err != nil {
		return nil, nil, err
	}

	nodes := make([]graphNode, 0, len(nodeOrder))
	for _, id := range nodeOrder {
		nodes = append(nodes, nodesByID[id])
	}
	edges := make([]graphEdge, 0, len(rels))
	seenEdge := map[string]bool{}
	for _, r := range rels {
		src, ok1 := eidToStable[r.startEid]
		dst, ok2 := eidToStable[r.endEid]
		if !ok1 || !ok2 {
			continue // endpoint not part of the returned subgraph
		}
		key := src + "|" + r.typ + "|" + dst
		if seenEdge[key] {
			continue
		}
		seenEdge[key] = true
		edges = append(edges, graphEdge{Source: src, Target: dst, Type: r.typ})
	}
	return nodes, edges, nil
}

// stableNodeID derives the stable string id for a node from its label and the
// natural key the schema constraints enforce (e.g. "User:nurse.joy@example.com",
// "Prompt:<uuid>", "Model:gemini", "Tool:jira").
func stableNodeID(n dbtype.Node) string {
	label := ""
	if len(n.Labels) > 0 {
		label = n.Labels[0]
	}
	keyProp := map[string]string{
		"User":      "email",
		"Model":     "provider",
		"Prompt":    "id",
		"Agent":     "id",
		"Tool":      "name",
		"Dataset":   "name",
		"Violation": "id",
	}[label]
	if keyProp != "" {
		if v, ok := n.Props[keyProp]; ok {
			return fmt.Sprintf("%s:%v", label, v)
		}
	}
	return fmt.Sprintf("%s:%s", label, n.ElementId)
}

// jsonValue converts neo4j driver values into JSON-marshalable ones.
func jsonValue(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = jsonValue(x)
		}
		return out
	case dbtype.Node:
		props := map[string]any{}
		for k, p := range t.Props {
			props[k] = jsonValue(p)
		}
		return props
	default:
		return v
	}
}
