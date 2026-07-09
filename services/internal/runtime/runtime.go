// Package runtime backs the telemetry-generator service. It produces SIMULATED
// Falco/Tetragon-style runtime-security events into an in-memory ring buffer.
//
// There is NO live kernel telemetry on the docker-compose demo path — Falco and
// Tetragon need eBPF and a real node (Brief R1). Every event this package emits
// carries Simulated=true and the SOC UI labels the feed SIMULATED. On the Helm
// hardening path (Phase 5, RUNTIME_MODE=ingest) the generator is off and REAL
// falcosidekick webhook alerts are normalized into the same ring buffer with
// Simulated=false (see falco.go).
package runtime

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// maxEvents is the ring-buffer capacity (keep the last ~200 events).
const maxEvents = 200

// Event is a SIMULATED runtime-security event
// (contracts/openapi/runtime.yaml RuntimeEvent).
type Event struct {
	EventID   string    `json:"eventId"`
	Source    string    `json:"source"`   // falco | tetragon
	Rule      string    `json:"rule"`     // e.g. "Terminal shell in container"
	Priority  string    `json:"priority"` // info | notice | warning | error | critical
	Message   string    `json:"message"`
	Namespace string    `json:"namespace,omitempty"`
	Pod       string    `json:"pod,omitempty"`
	Container string    `json:"container,omitempty"`
	Simulated bool      `json:"simulated"` // true: synthetic (compose path); false: real Falco (Helm ingest path)
	Timestamp time.Time `json:"timestamp"`
}

// rule pairs a Falco/Tetragon rule name with a default priority and a message
// template; the generator cycles through these deterministically.
type ruleSpec struct {
	rule     string
	priority string
	message  string
}

var ruleSpecs = []ruleSpec{
	{"Terminal shell in container", "warning", "A shell was spawned in a container (bash) — unexpected interactive access"},
	{"Sensitive file opened for reading", "warning", "Sensitive file below /etc opened for reading (/etc/shadow)"},
	{"Unexpected outbound connection", "notice", "Outbound connection to an unexpected destination on port 443"},
	{"Privilege escalation via setuid", "critical", "Setuid to root detected for a non-root process"},
	{"Package management launched in container", "error", "Package manager (apt) launched inside a running container"},
	{"Write below etc", "error", "File write attempted below /etc (/etc/cron.d)"},
	{"Crypto miner signature detected", "critical", "Process behavior matches a known crypto-miner signature (xmrig)"},
}

var sources = []string{"falco", "tetragon"}
var priorities = []string{"info", "notice", "warning", "error", "critical"}

var pods = []string{
	"prompt-svc-7d9f", "detection-svc-5b2c", "discovery-svc-9a41",
	"compliance-svc-3e8d", "opa-6c1a", "frontend-2f7b",
}

var containers = []string{
	"prompt-svc", "detection-svc", "discovery-svc", "compliance-svc", "opa", "frontend",
}

// Buffer is a thread-safe ring buffer of the last maxEvents events plus a
// deterministic generator driven by an incrementing counter.
type Buffer struct {
	mu      sync.Mutex
	events  []Event // newest appended at the tail; trimmed to maxEvents
	counter uint64  // increments per generated event; drives variation
}

// NewBuffer returns an empty ring buffer.
func NewBuffer() *Buffer {
	return &Buffer{events: make([]Event, 0, maxEvents)}
}

// Generate builds one SIMULATED event deterministically from the internal
// counter, appends it to the ring buffer (trimming to maxEvents), and returns it.
// Every event has Simulated=true and a UTC timestamp.
func (b *Buffer) Generate() Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	i := b.counter
	b.counter++

	spec := ruleSpecs[i%uint64(len(ruleSpecs))]
	// Mostly use the rule's natural priority, but vary occasionally so the feed
	// shows a spread of severities.
	priority := spec.priority
	if i%5 == 4 {
		priority = priorities[i%uint64(len(priorities))]
	}

	e := Event{
		EventID:   uuid.NewString(),
		Source:    sources[i%uint64(len(sources))],
		Rule:      spec.rule,
		Priority:  priority,
		Message:   spec.message,
		Namespace: "ai-sentinel",
		Pod:       pods[i%uint64(len(pods))],
		Container: containers[i%uint64(len(containers))],
		Simulated: true, // ALWAYS true — no live kernel telemetry (Brief R1)
		Timestamp: time.Now().UTC(),
	}

	b.events = append(b.events, e)
	if len(b.events) > maxEvents {
		b.events = b.events[len(b.events)-maxEvents:]
	}
	return e
}

// Append adds an externally ingested event (RUNTIME_MODE=ingest — normalized
// real Falco alerts) to the ring buffer, trimming to maxEvents.
func (b *Buffer) Append(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
	if len(b.events) > maxEvents {
		b.events = b.events[len(b.events)-maxEvents:]
	}
}

// List returns up to limit events, newest first, optionally filtered by source
// (falco|tetragon). limit defaults to 50 and is capped at maxEvents (200).
func (b *Buffer) List(source string, limit int) []Event {
	if limit <= 0 {
		limit = 50
	}
	if limit > maxEvents {
		limit = maxEvents
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]Event, 0, limit)
	// Walk from the tail (newest) backwards.
	for i := len(b.events) - 1; i >= 0 && len(out) < limit; i-- {
		e := b.events[i]
		if source != "" && e.Source != source {
			continue
		}
		out = append(out, e)
	}
	return out
}

// Len returns the current number of buffered events.
func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events)
}
