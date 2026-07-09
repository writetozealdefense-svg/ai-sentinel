// Command graph-svc is the Neo4j-backed risk graph service (Phase 3
// Interface 2). Port 8085, contract contracts/openapi/graph.yaml.
//
// On startup it applies the schema constraints (db/neo4j/schema.cypher,
// idempotent), backfills the graph from Postgres (source of truth, MERGE on
// natural keys so re-runs are idempotent), then subscribes to the NATS
// subjects for live incremental MERGEs (fail-soft if NATS is down).
//
// It exposes a fixed set of NAMED queries only — no arbitrary Cypher over the
// wire. Each response carries tabular rows, a renderable nodes/edges subgraph,
// and the exact Cypher executed.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/zealdefense/ai-sentinel/services/internal/bus"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
	"github.com/zealdefense/ai-sentinel/services/internal/httpx"
)

const serviceName = "graph-svc"

type server struct {
	driver neo4j.DriverWithContext
}

func main() {
	ctx := context.Background()

	databaseURL := httpx.Env(os.Getenv, "DATABASE_URL", "postgres://sentinel:sentinel_dev_pw@localhost:5432/sentinel?sslmode=disable")
	natsURL := httpx.Env(os.Getenv, "NATS_URL", "nats://localhost:4222")
	neo4jURL := httpx.Env(os.Getenv, "NEO4J_URL", "bolt://localhost:7687")
	neo4jUser := httpx.Env(os.Getenv, "NEO4J_USER", "neo4j")
	neo4jPassword := httpx.Env(os.Getenv, "NEO4J_PASSWORD", "sentinel_dev_pw")
	addr := ":" + httpx.Env(os.Getenv, "PORT", "8085")

	driver, err := neo4j.NewDriverWithContext(neo4jURL, neo4j.BasicAuth(neo4jUser, neo4jPassword, ""))
	if err != nil {
		log.Fatalf("[%s] neo4j driver: %v", serviceName, err)
	}
	defer driver.Close(ctx)

	// Neo4j can take a while to accept bolt connections after container start.
	if err := waitForNeo4j(ctx, driver, 60*time.Second); err != nil {
		log.Fatalf("[%s] neo4j not reachable at %s: %v", serviceName, neo4jURL, err)
	}
	log.Printf("[%s] connected to neo4j at %s", serviceName, neo4jURL)

	if err := applyConstraints(ctx, driver); err != nil {
		log.Fatalf("[%s] apply constraints: %v", serviceName, err)
	}
	log.Printf("[%s] schema constraints applied (idempotent)", serviceName)

	// Backfill from Postgres (source of truth). MERGE-only, so idempotent.
	database, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("[%s] db connect: %v", serviceName, err)
	}
	defer database.Close()
	if err := backfill(ctx, database, driver); err != nil {
		log.Fatalf("[%s] backfill: %v", serviceName, err)
	}

	// Live incremental population. bus.Connect is fail-soft: if NATS is down we
	// log and keep serving from the backfilled graph.
	b, _ := bus.Connect(natsURL, serviceName)
	defer b.Close()
	subscribeLive(ctx, b, driver)

	s := &server{driver: driver}

	r := httpx.NewRouter(serviceName)
	r.GET("/graph/queries", s.listQueries)
	r.POST("/graph/query/:name", s.runQuery)

	log.Printf("[%s] listening on %s", serviceName, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[%s] run: %v", serviceName, err)
	}
}

func waitForNeo4j(ctx context.Context, driver neo4j.DriverWithContext, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var err error
	for {
		vctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = driver.VerifyConnectivity(vctx)
		cancel()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		log.Printf("[%s] waiting for neo4j: %v", serviceName, err)
		time.Sleep(2 * time.Second)
	}
}

func (s *server) listQueries(c *gin.Context) {
	out := make([]gin.H, 0, len(namedQueryOrder))
	for _, name := range namedQueryOrder {
		q := namedQueries[name]
		out = append(out, gin.H{
			"name":        q.Name,
			"description": q.Description,
			"cypher":      q.Cypher,
			"parameters":  []string{}, // all named queries are parameterless
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *server) runQuery(c *gin.Context) {
	name := c.Param("name")
	q, ok := namedQueries[name]
	if !ok {
		httpx.Error(c, http.StatusNotFound, "not found", "unknown named query: "+name)
		return
	}
	res, err := q.execute(c.Request.Context(), s.driver)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, res)
}
