// Command telemetry-generator serves runtime-security events over HTTP for the
// SOC dashboard's runtime widgets. Port 8087. Two modes via RUNTIME_MODE:
//
//   - simulate (default, compose path): emits SIMULATED Falco/Tetragon-style
//     events into an in-memory ring buffer — there is NO live kernel telemetry
//     on the docker-compose demo path (Falco and Tetragon need eBPF and a real
//     node, Brief R1). EVERY event carries simulated:true and the UI labels the
//     feed SIMULATED. POST /runtime/events is rejected (409) in this mode.
//
//   - ingest (Helm hardening path, Phase 5): the internal generator is OFF and
//     POST /runtime/events accepts falcosidekick HTTP-webhook payloads,
//     normalizing them to RuntimeEvent with simulated:false.
package main

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zealdefense/ai-sentinel/services/internal/bus"
	"github.com/zealdefense/ai-sentinel/services/internal/httpx"
	"github.com/zealdefense/ai-sentinel/services/internal/runtime"
)

const serviceName = "telemetry-generator"

// subjectRuntimeObserved is the optional NATS subject for emitted runtime events.
const subjectRuntimeObserved = "ai.runtime.observed"

// RUNTIME_MODE values.
const (
	modeSimulate = "simulate"
	modeIngest   = "ingest"
)

// maxIngestBody caps the POST /runtime/events body at 1 MiB (413 beyond that).
const maxIngestBody = 1 << 20

type server struct {
	buf  *runtime.Buffer
	bus  *bus.Bus
	mode string // modeSimulate | modeIngest
}

func main() {
	addr := ":" + httpx.Env(os.Getenv, "PORT", "8087")

	mode := httpx.Env(os.Getenv, "RUNTIME_MODE", modeSimulate)
	if mode != modeSimulate && mode != modeIngest {
		log.Fatalf("[%s] invalid RUNTIME_MODE=%q (want %q or %q)", serviceName, mode, modeSimulate, modeIngest)
	}

	buf := runtime.NewBuffer()

	// Optional, fail-soft NATS publish. Only connect if NATS_URL is set so the
	// service runs cleanly standalone (no NATS dependency on the smoke path).
	var b *bus.Bus
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		b, _ = bus.Connect(natsURL, serviceName) // fail-soft
		defer b.Close()
	}

	s := &server{buf: buf, bus: b, mode: mode}

	if mode == modeSimulate {
		// Spawn the generator goroutine: one SIMULATED event every ~2–4s.
		go s.run()
	}

	r := s.router()

	log.Printf("[%s] listening on %s (RUNTIME_MODE=%s)", serviceName, addr, mode)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[%s] run: %v", serviceName, err)
	}
}

// router builds the Gin engine. /healthz is registered here rather than via
// httpx.NewRouter so it can expose the runtime mode alongside the shared
// {"status":"ok","service":...} shape.
func (s *server) router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName, "mode": s.mode})
	})
	r.GET("/runtime/events", s.listEvents)
	r.POST("/runtime/events", s.ingestEvents)
	return r
}

// run generates one event every ~2–4s and (if a bus is configured) publishes it
// to ai.runtime.observed. It never returns. Simulate mode only.
func (s *server) run() {
	// Seed a local RNG only for the inter-event delay; event CONTENT is driven by
	// a deterministic counter inside the buffer (not crypto randomness).
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		e := s.buf.Generate()
		s.publish(e)
		// 2–4 seconds.
		delay := 2*time.Second + time.Duration(rng.Intn(2001))*time.Millisecond
		time.Sleep(delay)
	}
}

// publish emits the event to NATS if a bus is configured; always fail-soft.
func (s *server) publish(e runtime.Event) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(subjectRuntimeObserved, map[string]any{
		"eventId":   e.EventID,
		"source":    e.Source,
		"rule":      e.Rule,
		"priority":  e.Priority,
		"message":   e.Message,
		"namespace": e.Namespace,
		"pod":       e.Pod,
		"container": e.Container,
		"simulated": e.Simulated,
		"timestamp": e.Timestamp.Format(time.RFC3339),
	})
}

// listEvents serves GET /runtime/events?limit=&source= — newest-first events.
// limit defaults to 50, max 200; source is an optional falco|tetragon filter.
func (s *server) listEvents(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	source := c.Query("source")
	if source != "" && source != "falco" && source != "tetragon" {
		source = "" // ignore invalid filter rather than error
	}
	events := s.buf.List(source, limit)
	if events == nil {
		events = []runtime.Event{}
	}
	c.JSON(http.StatusOK, events)
}

// ingestEvents serves POST /runtime/events (ingest mode only) — the
// falcosidekick HTTP-webhook endpoint. It accepts one JSON alert object per
// request (sidekick's behavior) or a JSON array of alerts, normalizes each to
// a RuntimeEvent with simulated:false, and appends them to the ring buffer.
//
// In simulate mode it returns 409 so the compose path cannot be polluted with
// fake "real" events.
func (s *server) ingestEvents(c *gin.Context) {
	if s.mode != modeIngest {
		httpx.Error(c, http.StatusConflict, "ingest disabled", "RUNTIME_MODE="+s.mode)
		return
	}

	if c.Request.ContentLength > maxIngestBody {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "payload too large",
			"body exceeds "+strconv.Itoa(maxIngestBody)+" bytes")
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxIngestBody+1))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "read body", err.Error())
		return
	}
	if len(body) > maxIngestBody {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "payload too large",
			"body exceeds "+strconv.Itoa(maxIngestBody)+" bytes")
		return
	}

	alerts, err := runtime.ParseFalcoPayload(body)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "malformed payload", err.Error())
		return
	}

	for _, a := range alerts {
		e := a.Normalize() // simulated:false — REAL Falco event
		s.buf.Append(e)
		s.publish(e)
	}
	c.JSON(http.StatusAccepted, gin.H{"accepted": len(alerts)})
}
