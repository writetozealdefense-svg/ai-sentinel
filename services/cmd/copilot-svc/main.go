// Command copilot-svc is the AI Security Copilot (Phase 4). Port 8086,
// contract contracts/openapi/copilot.yaml.
//
// Deterministic OFFLINE mode is the DEFAULT (Brief R3): every answer is a
// template composition over an evidence bundle retrieved from the platform's
// own data (violations, prompts, findings, agents, fixture controls) with
// citations. Live LLM mode is only reachable behind COPILOT_LIVE=true +
// COPILOT_API_KEY (internal/copilot.NewLiveClient), and even then citations
// stay deterministic.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zealdefense/ai-sentinel/services/internal/compliance"
	"github.com/zealdefense/ai-sentinel/services/internal/copilot"
	"github.com/zealdefense/ai-sentinel/services/internal/db"
	"github.com/zealdefense/ai-sentinel/services/internal/httpx"
)

const serviceName = "copilot-svc"

type server struct {
	ret  *copilot.Retriever
	live *copilot.LiveClient
}

func main() {
	ctx := context.Background()

	databaseURL := httpx.Env(os.Getenv, "DATABASE_URL", "postgres://sentinel:sentinel_dev_pw@localhost:5432/sentinel?sslmode=disable")
	fixturesDir := httpx.Env(os.Getenv, "COMPLIANCE_FIXTURES", "/fixtures/compliance")
	addr := ":" + httpx.Env(os.Getenv, "PORT", "8086")

	comp, err := compliance.Load(fixturesDir)
	if err != nil {
		log.Fatalf("[%s] load compliance fixtures from %s: %v", serviceName, fixturesDir, err)
	}
	log.Printf("[%s] loaded %d compliance frameworks from %s", serviceName, len(comp.Frameworks()), fixturesDir)

	database, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("[%s] db connect: %v", serviceName, err)
	}
	defer database.Close()

	live := copilot.NewLiveClient(os.Getenv)
	if live != nil {
		log.Printf("[%s] LIVE mode enabled (model %s)", serviceName, live.Model())
	} else {
		log.Printf("[%s] offline deterministic mode (default, Brief R3)", serviceName)
	}

	s := &server{
		ret:  &copilot.Retriever{DB: database, Comp: comp},
		live: live,
	}

	// Custom router (not httpx.NewRouter): the copilot healthz must report the
	// mode per contracts/openapi/copilot.yaml CopilotHealth.
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName, "mode": copilot.Mode(s.live)})
	})
	r.GET("/copilot/explain/:violationId", s.explain)
	r.POST("/copilot/ask", s.ask)

	log.Printf("[%s] listening on %s", serviceName, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[%s] run: %v", serviceName, err)
	}
}

// explain serves GET /copilot/explain/{violationId}: all three grounded views
// plus citations. 400 for a non-UUID id, 404 for an unknown one.
func (s *server) explain(c *gin.Context) {
	id := c.Param("violationId")
	if _, err := uuid.Parse(id); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request", "violationId must be a UUID")
		return
	}
	bundle, err := s.ret.EvidenceBundle(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(c, http.StatusNotFound, "not found", "no violation with id "+id)
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
		return
	}
	s.respondBundle(c, bundle, "", "all")
}

// askRequest matches contracts/openapi/copilot.yaml AskRequest.
type askRequest struct {
	Question    string `json:"question"`
	ViolationID string `json:"violationId"`
	View        string `json:"view"`
}

var validViews = map[string]bool{"": true, "all": true, "executive": true, "technical": true, "compliance": true}

// ask serves POST /copilot/ask. With violationId: same grounding as explain,
// optionally filtered to one view. Without: the deterministic intent router.
func (s *server) ask(c *gin.Context) {
	var req askRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	if req.Question == "" {
		httpx.Error(c, http.StatusBadRequest, "invalid request", "question is required")
		return
	}
	if !validViews[req.View] {
		httpx.Error(c, http.StatusBadRequest, "invalid request", "view must be one of all|executive|technical|compliance")
		return
	}

	ctx := c.Request.Context()

	if req.ViolationID != "" {
		if _, err := uuid.Parse(req.ViolationID); err != nil {
			httpx.Error(c, http.StatusBadRequest, "invalid request", "violationId must be a UUID")
			return
		}
		bundle, err := s.ret.EvidenceBundle(ctx, req.ViolationID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.Error(c, http.StatusNotFound, "not found", "no violation with id "+req.ViolationID)
				return
			}
			httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
			return
		}
		s.respondBundle(c, bundle, req.Question, req.View)
		return
	}

	// No violationId: deterministic intent router (never invents facts).
	switch copilot.Route(req.Question) {
	case copilot.IntentWhyBlocked:
		id, err := s.ret.MostRecentBlockViolationID(ctx)
		if err != nil {
			httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
			return
		}
		if id == "" {
			s.respondOffline(c, copilot.Views{Executive: "No violations with a BLOCK prompt decision are recorded."}, []copilot.Citation{}, req.View)
			return
		}
		bundle, err := s.ret.EvidenceBundle(ctx, id)
		if err != nil {
			httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
			return
		}
		s.respondBundle(c, bundle, req.Question, req.View)

	case copilot.IntentRiskiestUser:
		u, err := s.ret.RiskiestUser(ctx)
		if err != nil {
			httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
			return
		}
		if u == nil {
			s.respondOffline(c, copilot.Views{Executive: "No violations with an associated user are recorded."}, []copilot.Citation{}, req.View)
			return
		}
		views, cites := copilot.ComposeRiskiestUser(u)
		s.respondOffline(c, views, cites, req.View)

	case copilot.IntentRiskiestAgent:
		a, err := s.ret.RiskiestAgent(ctx)
		if err != nil {
			httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
			return
		}
		if a == nil {
			s.respondOffline(c, copilot.Views{Executive: "No agents are recorded."}, []copilot.Citation{}, req.View)
			return
		}
		views, cites := copilot.ComposeRiskiestAgent(a)
		s.respondOffline(c, views, cites, req.View)

	case copilot.IntentCompliancePosture:
		items, err := s.ret.CompliancePosture(ctx)
		if err != nil {
			httpx.Error(c, http.StatusInternalServerError, "query failed", err.Error())
			return
		}
		if len(items) == 0 {
			s.respondOffline(c, copilot.Views{Compliance: "No violations are recorded, so no compliance exposure is on file."}, []copilot.Citation{}, req.View)
			return
		}
		views, cites := copilot.ComposeCompliancePosture(items)
		s.respondOffline(c, views, cites, req.View)

	default: // IntentUnknown — honest fallback, empty-but-present citations.
		views, cites := copilot.ComposeFallback()
		s.respondOffline(c, views, cites, req.View)
	}
}

// respondBundle composes the deterministic views + citations from the bundle;
// in live mode it additionally asks the Claude Messages API (grounded on the
// same bundle), keeping the deterministic citations. A live failure falls back
// to the offline composition and reports mode offline (honest).
func (s *server) respondBundle(c *gin.Context, bundle *copilot.Bundle, question, view string) {
	views, cites := copilot.Compose(bundle)
	mode := "offline"
	if s.live != nil {
		liveViews, err := s.live.Explain(c.Request.Context(), question, bundle)
		if err != nil {
			log.Printf("[%s] live mode failed, falling back to offline composition: %v", serviceName, err)
		} else {
			views = liveViews
			mode = "live"
		}
	}
	c.JSON(http.StatusOK, copilot.AskResponse{Mode: mode, Views: views.Filter(view), Citations: cites})
}

// respondOffline emits a deterministic (router/aggregate/fallback) answer.
func (s *server) respondOffline(c *gin.Context, views copilot.Views, cites []copilot.Citation, view string) {
	if cites == nil {
		cites = []copilot.Citation{}
	}
	c.JSON(http.StatusOK, copilot.AskResponse{Mode: "offline", Views: views.Filter(view), Citations: cites})
}
