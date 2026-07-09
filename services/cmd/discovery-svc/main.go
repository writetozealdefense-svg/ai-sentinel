// Command discovery-svc records SIMULATED shadow-AI discovery events and serves
// the AI inventory (Interface 5). Port 8081.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zealdefense/ai-sentinel/services/internal/bus"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
	"github.com/zealdefense/ai-sentinel/services/internal/httpx"
)

const serviceName = "discovery-svc"

type server struct {
	db  *db.DB
	bus *bus.Bus
}

// discoveryEventDTO matches contracts/openapi/discover.yaml DiscoveryEvent.
type discoveryEventDTO struct {
	EventID      string    `json:"eventId,omitempty"`
	Provider     string    `json:"provider"`
	UserEmail    string    `json:"userEmail"`
	Source       string    `json:"source"`
	Action       string    `json:"action,omitempty"`
	ArtifactName string    `json:"artifactName,omitempty"`
	Bytes        int64     `json:"bytes,omitempty"`
	Simulated    bool      `json:"simulated"`
	Timestamp    time.Time `json:"timestamp"`
}

func main() {
	ctx := context.Background()

	databaseURL := httpx.Env(os.Getenv, "DATABASE_URL", "postgres://sentinel:sentinel_dev_pw@localhost:5432/sentinel?sslmode=disable")
	natsURL := httpx.Env(os.Getenv, "NATS_URL", "nats://localhost:4222")
	addr := ":" + httpx.Env(os.Getenv, "PORT", "8081")

	database, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("[%s] db connect: %v", serviceName, err)
	}
	defer database.Close()

	b, _ := bus.Connect(natsURL, serviceName) // fail-soft: continue if NATS is down
	defer b.Close()

	s := &server{db: database, bus: b}

	r := httpx.NewRouter(serviceName)
	r.POST("/discover/events", s.ingest)
	r.GET("/discover/events", s.list)
	r.GET("/discover/inventory", s.inventory)

	log.Printf("[%s] listening on %s", serviceName, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[%s] run: %v", serviceName, err)
	}
}

func (s *server) ingest(c *gin.Context) {
	var in discoveryEventDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	if in.Provider == "" || in.UserEmail == "" || in.Source == "" {
		httpx.Error(c, http.StatusBadRequest, "invalid request", "provider, userEmail, source are required")
		return
	}
	if in.Timestamp.IsZero() {
		in.Timestamp = time.Now().UTC()
	}
	// Simulated is always true on the compose path (Brief R4).
	in.Simulated = true

	id, evt := s.record(c.Request.Context(), in)
	if id == "" {
		httpx.Error(c, http.StatusInternalServerError, "persist failed", "could not record discovery event")
		return
	}
	_ = evt
	c.JSON(http.StatusAccepted, gin.H{"eventId": id, "accepted": true})
}

// record inserts the event and publishes ai.discovery.observed. Returns the new id.
func (s *server) record(ctx context.Context, in discoveryEventDTO) (string, discoveryEventDTO) {
	id, err := s.db.InsertDiscoveryEvent(ctx, db.DiscoveryEvent{
		Provider:     in.Provider,
		UserEmail:    in.UserEmail,
		Source:       in.Source,
		Action:       in.Action,
		ArtifactName: in.ArtifactName,
		Bytes:        in.Bytes,
		Simulated:    true,
		OccurredAt:   in.Timestamp,
	})
	if err != nil {
		log.Printf("[%s] insert discovery_event: %v", serviceName, err)
		return "", in
	}
	in.EventID = id

	// Publish ai.discovery.observed (schema: discovery.observed.schema.json).
	s.bus.Publish(bus.SubjectDiscoveryObserved, map[string]any{
		"eventId":      id,
		"provider":     in.Provider,
		"userEmail":    in.UserEmail,
		"source":       in.Source,
		"action":       in.Action,
		"artifactName": in.ArtifactName,
		"bytes":        in.Bytes,
		"simulated":    true,
		"timestamp":    in.Timestamp.Format(time.RFC3339),
	})
	return id, in
}

func (s *server) list(c *gin.Context) {
	provider := c.Query("provider")
	limit := atoiDefault(c.Query("limit"), 50)
	events, err := s.db.ListDiscoveryEvents(c.Request.Context(), provider, limit)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	out := make([]discoveryEventDTO, 0, len(events))
	for _, e := range events {
		out = append(out, discoveryEventDTO{
			EventID:      e.ID,
			Provider:     e.Provider,
			UserEmail:    e.UserEmail,
			Source:       e.Source,
			Action:       e.Action,
			ArtifactName: e.ArtifactName,
			Bytes:        e.Bytes,
			Simulated:    e.Simulated,
			Timestamp:    e.OccurredAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

type inventoryItemDTO struct {
	Provider    string     `json:"provider"`
	UsageCount  int        `json:"usageCount"`
	UniqueUsers int        `json:"uniqueUsers"`
	Sanctioned  bool       `json:"sanctioned"`
	FirstSeen   *time.Time `json:"firstSeen,omitempty"`
	LastSeen    *time.Time `json:"lastSeen,omitempty"`
}

func (s *server) inventory(c *gin.Context) {
	items, err := s.db.Inventory(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	out := make([]inventoryItemDTO, 0, len(items))
	for _, it := range items {
		first, last := it.FirstSeen, it.LastSeen
		out = append(out, inventoryItemDTO{
			Provider:    it.Provider,
			UsageCount:  it.UsageCount,
			UniqueUsers: it.UniqueUsers,
			Sanctioned:  it.Sanctioned,
			FirstSeen:   &first,
			LastSeen:    &last,
		})
	}
	c.JSON(http.StatusOK, out)
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
