// Package bus is a thin NATS publish/subscribe wrapper. It is intentionally
// fail-soft: if NATS is unreachable the constructor returns a Bus whose Publish
// is a logged no-op, so the inspect path never crashes when NATS is down.
package bus

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

// Subjects (contracts/events/SUBJECTS.md).
const (
	SubjectDiscoveryObserved = "ai.discovery.observed"
	SubjectPromptInspected   = "ai.prompt.inspected"
	SubjectViolationRaised   = "ai.violation.raised"
	SubjectAgentActivity     = "ai.agent.activity"
)

// Bus wraps a NATS connection. A nil/disconnected conn degrades to no-op publishes.
type Bus struct {
	nc      *nats.Conn
	service string
}

// Connect dials NATS at url. On failure it logs and returns a degraded Bus
// (no connection) plus the error, so callers can `log + continue`.
func Connect(url, service string) (*Bus, error) {
	nc, err := nats.Connect(url,
		nats.Name(service),
		nats.MaxReconnects(-1),
		nats.RetryOnFailedConnect(true),
	)
	if err != nil {
		log.Printf("[bus] NATS connect failed (%s): %v — continuing without bus", url, err)
		return &Bus{service: service}, err
	}
	log.Printf("[bus] connected to NATS at %s", url)
	return &Bus{nc: nc, service: service}, nil
}

// Publish marshals v to JSON and publishes it on subject. Errors are logged and
// swallowed so a publish failure never breaks the request path.
func (b *Bus) Publish(subject string, v any) {
	if b == nil || b.nc == nil {
		log.Printf("[bus] skip publish to %s (no NATS connection)", subject)
		return
	}
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("[bus] marshal for %s failed: %v", subject, err)
		return
	}
	if err := b.nc.Publish(subject, data); err != nil {
		log.Printf("[bus] publish to %s failed: %v", subject, err)
		return
	}
}

// Subscribe registers handler for subject. No-op if there is no connection.
func (b *Bus) Subscribe(subject string, handler func([]byte)) (*nats.Subscription, error) {
	if b == nil || b.nc == nil {
		log.Printf("[bus] skip subscribe to %s (no NATS connection)", subject)
		return nil, nil
	}
	return b.nc.Subscribe(subject, func(m *nats.Msg) { handler(m.Data) })
}

// Close drains and closes the connection if present.
func (b *Bus) Close() {
	if b != nil && b.nc != nil {
		_ = b.nc.Drain()
		b.nc.Close()
	}
}
