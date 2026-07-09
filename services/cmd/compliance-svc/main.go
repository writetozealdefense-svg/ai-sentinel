// Command compliance-svc serves a read API over the compliance fixtures and the
// violations table (Interface 4). Port 8084.
//
// Brief R5: control IDs come ONLY from fixtures; this service never invents them.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zealdefense/ai-sentinel/services/internal/bus"
	"github.com/zealdefense/ai-sentinel/services/internal/compliance"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
	"github.com/zealdefense/ai-sentinel/services/internal/httpx"
)

const serviceName = "compliance-svc"

// violationPenalty is the per-open-violation score deduction. Score = clamp(100 - penalty*open, 0, 100).
const violationPenalty = 10

type server struct {
	db  *db.DB
	set *compliance.Set
}

func main() {
	ctx := context.Background()

	databaseURL := httpx.Env(os.Getenv, "DATABASE_URL", "postgres://sentinel:sentinel_dev_pw@localhost:5432/sentinel?sslmode=disable")
	natsURL := httpx.Env(os.Getenv, "NATS_URL", "nats://localhost:4222")
	fixturesDir := httpx.Env(os.Getenv, "COMPLIANCE_FIXTURES", "/fixtures/compliance")
	addr := ":" + httpx.Env(os.Getenv, "PORT", "8084")

	set, err := compliance.Load(fixturesDir)
	if err != nil {
		log.Fatalf("[%s] load fixtures from %s: %v", serviceName, fixturesDir, err)
	}
	log.Printf("[%s] loaded %d frameworks from %s", serviceName, len(set.Frameworks()), fixturesDir)

	database, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("[%s] db connect: %v", serviceName, err)
	}
	defer database.Close()

	// Optional live cache subscription; the violations table remains source of truth.
	b, _ := bus.Connect(natsURL, serviceName)
	defer b.Close()
	_, _ = b.Subscribe(bus.SubjectViolationRaised, func(data []byte) {
		log.Printf("[%s] observed ai.violation.raised (%d bytes)", serviceName, len(data))
	})

	s := &server{db: database, set: set}

	r := httpx.NewRouter(serviceName)
	r.GET("/compliance/frameworks", s.frameworks)
	r.GET("/compliance/frameworks/:framework/controls", s.controls)
	r.GET("/compliance/violations", s.violations)

	log.Printf("[%s] listening on %s", serviceName, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[%s] run: %v", serviceName, err)
	}
}

type frameworkScoreDTO struct {
	Framework      string `json:"framework"`
	DisplayName    string `json:"displayName"`
	Score          int    `json:"score"`
	ControlCount   int    `json:"controlCount"`
	OpenViolations int    `json:"openViolations"`
	ReviewRequired bool   `json:"reviewRequired"`
}

func (s *server) frameworks(c *gin.Context) {
	ctx := c.Request.Context()
	out := make([]frameworkScoreDTO, 0)
	for _, fw := range s.set.Frameworks() {
		open, err := s.db.CountOpenViolations(ctx, fw.Framework)
		if err != nil {
			httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
			return
		}
		// reviewRequired if any loaded control still carries its review marker.
		review := false
		for _, ctrl := range fw.Controls {
			if compliance.IsReview(ctrl) {
				review = true
				break
			}
		}
		score := 100 - violationPenalty*open
		if score < 0 {
			score = 0
		}
		out = append(out, frameworkScoreDTO{
			Framework:      fw.Framework,
			DisplayName:    fw.DisplayName,
			Score:          score,
			ControlCount:   len(fw.Controls),
			OpenViolations: open,
			ReviewRequired: review,
		})
	}
	c.JSON(http.StatusOK, out)
}

type controlMappingDTO struct {
	Framework        string   `json:"framework"`
	ControlID        string   `json:"controlId"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	MappedCategories []string `json:"mappedCategories"`
	Severity         string   `json:"severity"`
	ReviewMarker     string   `json:"reviewMarker"`
}

func (s *server) controls(c *gin.Context) {
	name := c.Param("framework")
	fw, ok := s.set.Framework(name)
	if !ok {
		httpx.Error(c, http.StatusNotFound, "not found", "unknown framework: "+name)
		return
	}
	out := make([]controlMappingDTO, 0, len(fw.Controls))
	for _, ctrl := range fw.Controls {
		out = append(out, controlMappingDTO{
			Framework:        fw.Framework,
			ControlID:        ctrl.ControlID,
			Title:            ctrl.Title,
			Description:      ctrl.Description,
			MappedCategories: ctrl.MappedCategories,
			Severity:         ctrl.Severity,
			ReviewMarker:     ctrl.Review, // verbatim review string from the fixture
		})
	}
	c.JSON(http.StatusOK, out)
}

type violationDTO struct {
	ViolationID string    `json:"violationId"`
	PromptID    string    `json:"promptId,omitempty"`
	UserEmail   string    `json:"userEmail,omitempty"`
	Framework   string    `json:"framework"`
	ControlID   string    `json:"controlId"`
	Category    string    `json:"category"`
	Severity    string    `json:"severity"`
	Description string    `json:"description,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

func (s *server) violations(c *gin.Context) {
	framework := c.Query("framework")
	limit := atoiDefault(c.Query("limit"), 50)
	rows, err := s.db.ListViolations(c.Request.Context(), framework, limit)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	out := make([]violationDTO, 0, len(rows))
	for _, v := range rows {
		out = append(out, violationDTO{
			ViolationID: v.ID,
			PromptID:    v.PromptID,
			UserEmail:   v.UserEmail,
			Framework:   v.Framework,
			ControlID:   v.ControlID,
			Category:    v.Category,
			Severity:    v.Severity,
			Description: v.Description,
			Timestamp:   v.OccurredAt,
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
